package dola

import "time"

// StatefulLogger upgrades TRACE log lines into INFO every once in a while.
type StatefulLogger struct {
	traceEvery time.Duration
	traceLast  time.Time
}

func NewTimedLogger() StatefulLogger {
	return StatefulLogger{
		traceEvery: 0,
		traceLast:  time.Time{},
	}
}

func (t *StatefulLogger) TraceEvery(x time.Duration) {
	t.traceEvery = x
}
