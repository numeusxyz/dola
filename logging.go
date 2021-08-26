package dola

import (
	"sync/atomic"
	"time"
)

const (
	// In dormant mode the StatefulLogger outputs trace logs as info every once in a
	// while.
	Dormant = iota
	// In operational mode everything is outputted as-is.
	Operational = iota
)

type StatefulLogger struct {
	state int32

	traceEvery time.Duration
	traceLast  time.Time
}

func NewTimedLogger() StatefulLogger {
	return StatefulLogger{
		state:      Dormant,
		traceEvery: 0,
		traceLast:  time.Time{},
	}
}

// WakeUp gets the StatefulLogger out of the dormant state for a predefined amount of
// time.
func (t *StatefulLogger) WakeUp() {
	atomic.CompareAndSwapInt32(&t.state, Dormant, Operational)
}
