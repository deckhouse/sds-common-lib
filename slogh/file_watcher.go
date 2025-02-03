package slogh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"time"
	"unsafe"

	"github.com/fsnotify/fsnotify"
)

const configFileSizeLimit = 1 << 15 /* 32KiB */

var errWatcherSubscriptionLost = errors.New("file watcher subscription was removed")

var errConfigRead = errors.New("unable to read config")

var errConfigProcess = errors.New("unable to process config file")

type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(d time.Duration)
}

var _ Ticker = &ticker{}

type ticker struct {
	*time.Ticker
}

// C implements Ticker.
func (t *ticker) C() <-chan time.Time {
	return t.Ticker.C
}

type ConfigFileWatcherOptions struct {
	// Where watcher's own logs should go. If nil, [slog.Default] will be used
	OwnLogger *slog.Logger
	// How much to wait before retrying critical errors like missing file.
	// Default is 10s.
	RetryInterval *time.Duration
	// Maximum rate at which updates will be sent to [UpdateConfigDataFunc].
	// Duplicates will be "merged" and sent later. Default is 1s.
	DedupInterval *time.Duration
	// For testing purposes.
	NewTicker func(d time.Duration) Ticker
}

type UpdateConfigDataFunc func(data map[string]string) error

// TODO default filePath="./slogh.cfg"
// TODO also support overriding with SLOGH_CONFIG_PATH
// TODO(optional) sac reload latency to avoid duplicate reload (after test in k8s)
// TODO: block on the initial recload, continue in case of errors
func RunConfigFileWatcher(
	ctx context.Context,
	filePath string,
	update UpdateConfigDataFunc,
	opts *ConfigFileWatcherOptions,
) {
	var log *slog.Logger

	// don't crash the app
	defer func() {
		if r := recover(); r != nil {
			if log != nil {
				log.Error("panic recovered", "err", r, "stack", debug.Stack())
			} else {
				fmt.Printf("panic recovered: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}
	}()

	// own logger
	if opts != nil {
		if log = opts.OwnLogger; log == nil {
			// TODO write to same log?
			log = slog.Default()
		}
	}

	// polling loop, which should normally be replaced with loop in [watchConfig]
	retryInterval := time.Second * 10
	if opts != nil && opts.RetryInterval != nil {
		retryInterval = *opts.RetryInterval
	}

	// deduplication: reload config no more then once per [dedupInterval]
	dedupInterval := time.Second * 1
	if opts != nil && opts.DedupInterval != nil {
		dedupInterval = *opts.DedupInterval
	}

	// ticker
	newTicker := func(d time.Duration) Ticker {
		return &ticker{time.NewTicker(d)}
	}
	if opts != nil && opts.NewTicker != nil {
		newTicker = opts.NewTicker
	}

	log.Info("config file watcher started", "file", filePath)
	defer func() {
		log.Info("config file watcher stopped", "file", filePath)
	}()

	for {
		if ctx.Err() != nil {
			log.Debug("finished reloading config")
			return
		}

		if err := reloadConfig(filePath, update, log); err != nil {
			log.Error("periodic config reload failed", "err", err)
		} else if err := watchConfig(
			ctx,
			filePath,
			update,
			log,
			newTicker,
			dedupInterval,
		); err != nil {
			if errors.Is(err, errWatcherSubscriptionLost) {
				log.Debug("subscription lost: reloading watcher immediately")
			} else {
				log.Error("watching config file failed", "err", err)
			}
		}

		// error branch: want to wait before repeat
		time.Sleep(retryInterval)
	}
}

func watchConfig(
	ctx context.Context,
	filePath string,
	update UpdateConfigDataFunc,
	log *slog.Logger,
	newTicker func(d time.Duration) Ticker,
	dedupInterval time.Duration,
) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating file watcher: %w", err)
	}

	defer fw.Close()

	if err := fw.Add(filePath); err != nil {
		return fmt.Errorf("adding file to watchlist: %w", err)
	}

	log.Debug("started watching 'file'", "file", filePath)

	var lastReload time.Time

	// duplicate events will raise [missedEvents] flag
	var missedEvents bool

	// to flush [missedEvents], and
	// to monitor lost subsriptions, due to file removal
	statusTicker := newTicker(dedupInterval)
	defer statusTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("finished watching 'file'", "file", filePath)
			return nil
		case <-statusTicker.C():
			if len(fw.WatchList()) == 0 {
				// path was removed (e.g. due to file move) -> want watcher reload
				return errWatcherSubscriptionLost
			}
			if !missedEvents {
				continue
			}
			missedEvents = false
		case event := <-fw.Events:
			log.Debug("received filesystem event for 'file': 'op'", "file", event.Name, "op", event.Op.String())
			if !event.Has(fsnotify.Write) {
				continue
			}
			if time.Since(lastReload) < dedupInterval {
				missedEvents = true
				continue
			}
			statusTicker.Reset(dedupInterval)
		case err := <-fw.Errors:
			// syscall failure -> want watcher reload
			return fmt.Errorf("error event: %w", err)
		}

		if err := reloadConfig(filePath, update, log); errors.Is(err, errConfigRead) {
			// permissions, missing file, etc -> want watcher reload
			return fmt.Errorf("reloading config on watch event: %w", err)
		} else {
			lastReload = time.Now()
			if err != nil {
				// file format, too big file, etc -> keep watching
				log.Error("error during file reload on watch event", "err", err)
			}
		}
	}
}

func reloadConfig(filePath string, update UpdateConfigDataFunc, log *slog.Logger) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("%w: %w", errConfigRead, err)
	}

	defer file.Close()

	fileBytes, err := io.ReadAll(io.LimitReader(file, int64(configFileSizeLimit)))
	if err != nil {
		return fmt.Errorf("%w: %w", errConfigRead, err)
	}

	if len(fileBytes) == configFileSizeLimit {
		return fmt.Errorf(
			"%w: config file limit reached (%d), reload is not supported",
			errConfigProcess,
			configFileSizeLimit,
		)
	}

	// format is: '='-separated keys and values, each pair on a separate line
	lines := bytes.Split(fileBytes, []byte{'\n'})

	cfgData := make(map[string]string, len(lines))

	for i, line := range lines {
		key, value, found := bytes.Cut(line, []byte{'='})

		if !found {
			log.Debug(
				"skipping line 'line', since there's no `=` sign",
				"line", i+1,
			)
			continue
		}

		// TODO bug - trim before quoting?
		key = bytes.TrimSpace(key)
		value = bytes.TrimSpace(value)

		keyStr := unsafe.String(unsafe.SliceData(key), len(key))
		valueStr := unsafe.String(unsafe.SliceData(value), len(value))

		if value[0] == '"' || value[0] == '`' {
			// Go string literal syntax
			valueStr, err = strconv.Unquote(valueStr)
			if err != nil {
				return fmt.Errorf(
					"%w: line %d entered string literal mode, but syntax is incorrect: %w",
					errConfigProcess,
					i+1,
					err,
				)
			}
		}

		cfgData[keyStr] = valueStr
	}

	if err := update(cfgData); err != nil {
		return fmt.Errorf("%w: updating config data file: %w", errConfigProcess, err)
	}

	log.Info("reloaded config", "cfgData", cfgData, "file", filePath)

	return nil
}
