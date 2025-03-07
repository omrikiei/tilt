package engine

import (
	"github.com/windmilleng/tilt/internal/store"
)

type tiltfileLogWriter struct {
	store store.RStore
}

func NewTiltfileLogWriter(s store.RStore) *tiltfileLogWriter {
	return &tiltfileLogWriter{s}
}

func (w *tiltfileLogWriter) Write(p []byte) (n int, err error) {
	w.store.Dispatch(TiltfileLogAction{
		LogEvent: store.NewGlobalLogEvent(p),
	})
	return len(p), nil
}
