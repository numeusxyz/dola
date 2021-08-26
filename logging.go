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
	// In dormant mode the AwakenLogger outputs trace logs as info
	// every once in a while.
	Dormant = iota
	// In awaken mode everything is outputted as-is.
	Awaken = iota
)

// LogState keeps track of what the current state is.
type LogState struct {
	state int32
	// For how long state should be kept Awaken.
	duration time.Duration
}

func NewLogState(duration time.Duration) LogState {
	return LogState{
		state:    Dormant,
		duration: duration,
	}
}

func (s *LogState) WakeUp() {
	if atomic.CompareAndSwapInt32(&s.state, Dormant, Awaken) {
		time.AfterFunc(s.duration, func() {
			if !atomic.CompareAndSwapInt32(&s.state, Awaken, Dormant) {
				panic("illegal state")
			}
		})
	}
}

func (s *LogState) Awaken() bool {
	return atomic.LoadInt32(&s.state) == Awaken
}

// +----------------+
// | AwakenLogger |
// +----------------+

type AwakenLogger struct {
	state LogState

	traceEvery time.Duration
	traceLast  time.Time
}

func NewAwakenLogger(d time.Duration) AwakenLogger {
	// We use the same duration for both the time awaken and how
	// often trace logs should be allowed.
	return AwakenLogger{
		state:      NewLogState(d),
		traceEvery: d,
		traceLast:  time.Time{},
	}
}

// WakeUp gets the AwakenLogger out of its dormant state for a an
// amount of time.
func (t *AwakenLogger) WakeUp() {
	t.state.WakeUp()
}

func (t *AwakenLogger) Trace() *zerolog.Event {
	if t.state.Awaken() {
		return log.Info()
	}

	return t.dormantTrace()
}

// dormantTrace is left unlocked on purpose.  If there is a race
// condition and more than one thread set `traceLast` to Now(), we
// don't care, it's still Now().
func (t *AwakenLogger) dormantTrace() *zerolog.Event {
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
