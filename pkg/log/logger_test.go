package log

import "testing"

func TestBasic(t *testing.T) {

	log := New("log")

	log.Verbose("hello", "world")
	log.Verbosef("%s %s %d", "hello", "world", 2)

	logWithLocation := NewWithOpts("logWithLocation", Options{CallerLocation: true})
	logWithLocation.Info("here")

}
