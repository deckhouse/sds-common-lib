package u

import "log/slog"

// LogError logs err using log if it is non-nil and returns err unchanged.
func LogError(log *slog.Logger, err error) error {
	if err != nil {
		log.Error(err.Error())
	}
	return err
}
