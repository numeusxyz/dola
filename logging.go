package dola

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// +------------------+
// | Stateful logging |
// +------------------+

const (
	// In dormant mode the StatefulLogger outputs trace logs as info every once in a
	// while.
	Dormant = iota
	// In operational mode everything is outputted as-is.
	Operational = iota
)

type StatefulLogger struct {
	state int32

	traceLast  time.Time
	traceEvery time.Duration

	awakenAt time.Time

	m sync.Mutex
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

func (t *StatefulLogger) Trace() *zerolog.Event {
	switch t.state {
	case Dormant:
		return t.dormantTrace()
	case Operational:
		return log.Trace()
	}
	panic("illegal state")
}

func (t *StatefulLogger) dormantTrace() *zerolog.Event {
	if time.Since(t.traceLast) < t.traceEvery {
		t.traceLast = time.Now()
		return log.Trace()
	}

	return log.Trace().Discard()
}

// +-------------------+
// | Stateless logging |
// +-------------------+

func Code(e *zerolog.Event, code string) {
	if code != "" {
		e = e.Str("code", code)
	}

	e.Msg(Location2())
}

func What(e *zerolog.Event, what string) {
	if what != "" {
		e = e.Str("what", what)
	}

	e.Msg(Location2())
}

func Msg(e *zerolog.Event) {
	e.Msg(Location2())
}
