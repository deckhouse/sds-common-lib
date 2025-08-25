package utils

import (
	"errors"
	"fmt"
)

// ErrRecoveredFromPanic is a sentinel error that indicates a panic was recovered
var ErrRecoveredFromPanic = errors.New("recovered from panic")

// RecoverPanicToErr recovers from a panic and joins it into err together with
// ErrRecoveredFromPanic.
func RecoverPanicToErr(err *error) {
	v := recover()
	if v == nil {
		return
	}

	var verr error
	switch vt := v.(type) {
	case string:
		verr = errors.New(vt)
	case error:
		verr = vt
	default:
		verr = errors.New(fmt.Sprint(v))
	}

	*err = errors.Join(*err, ErrRecoveredFromPanic, verr)
}
