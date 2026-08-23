package utils

// ErrorLogger is the one method LogError needs. Both *slog.Logger and the
// logger.Logger of this module satisfy it, so a caller does not have to unwrap
// one into the other to report an error.
type ErrorLogger interface {
	Error(msg string, args ...any)
}

// LogError logs err using log if it is non-nil and returns err unchanged.
func LogError(log ErrorLogger, err error) error {
	if err != nil {
		log.Error(err.Error())
	}
	return err
}
