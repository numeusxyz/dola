package dola

import (
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// +----------+
// | LogState |
// +----------+

const (
	// In dormant mode the StatefulLogger outputs trace logs as info every once in a
	// while.
	Dormant = iota
	// In awaken mode everything is outputted as-is.
	Awaken = iota
)

type LogState struct {
	state int32
	// Time of last awakening.
	awakenAt time.Time
	// For how long state should be kept Awaken.
	duration time.Duration
}

func NewLogState(duration time.Duration) LogState {
	return LogState{
		state:    Dormant,
		awakenAt: time.Time{},
		duration: duration,
	}
}

func (s *LogState) WakeUp() {
	if atomic.CompareAndSwapInt32(&s.state, Dormant, Awaken) {
		s.awakenAt = time.Now()
	}
}

func (s *LogState) Awaken() bool {
	if time.Since(s.awakenAt) < s.duration {
		return true
	}

	// Go back into dormant state, if not already.
	atomic.CompareAndSwapInt32(&s.state, Awaken, Dormant)

	return false
}

// +----------------+
// | StatefulLogger |
// +----------------+

type StatefulLogger struct {
	state LogState

	traceEvery time.Duration
	traceLast  time.Time
}

func NewStatefulLogger(d time.Duration) StatefulLogger {
	// We use the same duration for both the time awaken and how
	// often trace logs should be allowed.
	return StatefulLogger{
		state:      NewLogState(d),
		traceEvery: d,
		traceLast:  time.Time{},
	}
}

// WakeUp gets the StatefulLogger out of its dormant state for a an
// amount of time.
func (t *StatefulLogger) WakeUp() {
	t.state.WakeUp()
}

func (t *StatefulLogger) Trace() *zerolog.Event {
	if t.state.Awaken() {
		return log.Info()
	}

	return t.dormantTrace()
}

// dormantTrace is left unlocked on purpose.  If there is a race
// condition and more than one thread set `traceLast` to Now(), we
// don't care, it's still Now().
func (t *StatefulLogger) dormantTrace() *zerolog.Event {
	if time.Since(t.traceLast) > t.traceEvery {
		t.traceLast = time.Now()

		return log.Info()
	}

	return log.Trace()
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
