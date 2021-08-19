package dola

import (
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

// Location returns the name of the parent calling function.
func Location() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "?"
	}

	fn := runtime.FuncForPC(pc)
	xs := strings.SplitAfterN(fn.Name(), "/", 3) // nolint: gomnd

	return xs[len(xs)-1]
}

// Location2 returns the name of the grandparent calling function.
func Location2() string {
	pc, _, _, ok := runtime.Caller(2) // nolint:gomnd
	if !ok {
		return "?"
	}

	fn := runtime.FuncForPC(pc)
	xs := strings.SplitAfterN(fn.Name(), "/", 3) // nolint: gomnd

	return xs[len(xs)-1]
}

// +---------+
// | Checker |
// +---------+

// Checker is a simple tool to check if everything initialized is
// subsequently deinitialized.  Works from simple open/close calls to
// gourintes.
type resourceChecker struct {
	m         sync.Mutex
	resources map[string]int
}

// nolint:gochecknoglobals
var defaultResourceChecker = resourceChecker{
	m:         sync.Mutex{},
	resources: make(map[string]int),
}

func CheckerPush(xs ...string) {
	var name string

	switch len(xs) {
	case 0:
		name = Location2()
	case 1:
		name = xs[0]
	default:
		panic("invalid argument")
	}

	defaultResourceChecker.m.Lock()
	defaultResourceChecker.resources[name]++
	defaultResourceChecker.m.Unlock()
}

func CheckerPop(xs ...string) {
	var name string

	switch len(xs) {
	case 0:
		name = Location2()
	case 1:
		name = xs[0]
	default:
		panic("invalid argument")
	}

	defaultResourceChecker.m.Lock()
	defaultResourceChecker.resources[name]--
	defaultResourceChecker.m.Unlock()
}

// CheckerAssert should be defer-called in main().
func CheckerAssert() {
	Msg(log.Debug(), "checking resources...", "")
	time.Sleep(1 * time.Second)

	defaultResourceChecker.m.Lock()
	defer defaultResourceChecker.m.Unlock()

	for k, v := range defaultResourceChecker.resources {
		if v != 0 {
			Msg(log.Warn().Int("counter", v).Str("unit", k), "leaked resource", "")
		}
	}
}

// +---------+
// | Logging |
// +---------+

func Msg(e *zerolog.Event, what, code string) {
	if what != "" {
		e = e.Str("what", what)
	}

	if code != "" {
		e = e.Str("code", code)
	}

	e.Msg(Location2())
}

// +----------------+
// | ErrorWaitGroup |
// +----------------+

type ErrorWaitGroup struct {
	err   error
	group sync.WaitGroup
	mutex sync.Mutex
}

func NewErrorWaitGroup(initial error) *ErrorWaitGroup {
	return &ErrorWaitGroup{
		err:   initial,
		group: sync.WaitGroup{},
		mutex: sync.Mutex{},
	}
}

func (m *ErrorWaitGroup) Add(delta int) {
	m.group.Add(delta)
}

func (m *ErrorWaitGroup) Done(right error) {
	m.mutex.Lock()
	m.err = multierr.Append(m.err, right)
	m.mutex.Unlock()
	m.group.Done()
}

func (m *ErrorWaitGroup) Wait() error {
	m.group.Wait()
	return m.err
}

// +----------+
// | Profiler |
// +----------+

type Profiler struct {
	Filename string
}

func NewProfiler(filename string) Profiler {
	p := Profiler{filename}

	f, err := os.Create(p.Filename)
	if err != nil {
		panic(err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		panic(err)
	}

	return p
}

func (p Profiler) Stop() {
	pprof.StopCPUProfile()
}
