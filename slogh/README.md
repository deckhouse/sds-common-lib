# slogh

This is a standard Go `slog.Handler` interface implementation that includes features and defaults helpful for the Deckhouse ecosystem.

# Usage

It's usable via any logging interface, which supports writing to `slog.Handler`.

## slog.Logger

```go
import (
	"log/slog"

	"github.com/deckhouse/sds-common-lib/slogh"
)

func main() {
	log := slog.New(&slogh.Handler{})

	log.Info("slog is intended to replace the standard log", "v", "1.23.4")
}
```

## logr.Logger

This is popular logging interface, which is also used by `sigs.k8s.io/controller-runtime`. Here's how to integrate it:

``` go
import (
	"github.com/deckhouse/sds-common-lib/slogh"
	"github.com/go-logr/logr"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
)

func main() {
	log := logr.FromSlogHandler(&slogh.Handler{})

	crlog.SetLogger(log)

	log.V(0).Info("logr is widely used in k8s", "v", "1.4.2")
}
```

# Features

## Config file with automatic reload

``` go
slogh.EnableConfigReload(context.Background(), nil)
log := slog.New(&slogh.Handler{})
```

`slogh.EnableConfigReload` will load config (`./slogh.cfg` by default) and start a resilient background file watcher, which reloads the configuration on each change. Configuration lives in the package, so every handler picks the change up -- pass a `*slogh.ConfigFileWatcherOptions` to override the path, the retry interval or where the watcher logs its own errors. Configuration file has a simple key=value format:

```
# lines without equal sign are ignored
# those are all keys with default values:

# any slog level, or just a number
level=INFO

# also supported: "text"
format=json

# for each log print "source" property with information about callsite
callsite=true

render=true
stringValues=true
```

Alternative configuration file location can be provided directly with `slogh.ConfigFileWatcherOptions` (higher priority), or with env var `SLOGH_CONFIG_PATH` (lower priority).

In Kubernetes, you can map `ConfigMap` into a config file in your container, and it will be reloaded automatically without container restart. See [instruction](https://kubernetes.io/docs/concepts/storage/volumes/#configmap).

## Token rendering in messages

Option `render=true` or `slogh.Config{Render: slogh.RenderEnabled}` allows to render attribute values directly to your messages, using single-quoted attribute names as tokens.

For example, this log:

```go
log.Info("received request from 'user'", "user", user.Name)
```
Will produce message like this:
```
{<...> "msg":"received request from alice","user":"alice"}
```

## Stringing JSON values

Usually it's easier to parse JSON, which has all values stringed (e.g. `true` is `"true"`, `123.4` is `"123.4"`). Therefore, the default value for `stringValues=true`.

```go
log.Info("some message", "loggedIn", true)
```
Will produce value like this:
```
{<...> ,"loggedIn":"true"}
```
