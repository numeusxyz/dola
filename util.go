package dola

import (
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

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
	log.Debug().Str("what", "Checking resources...").Msg(Location())
	time.Sleep(1 * time.Second)

	defaultResourceChecker.m.Lock()
	defer defaultResourceChecker.m.Unlock()

	for k, v := range defaultResourceChecker.resources {
		if v != 0 {
			log.Warn().Int("counter", v).Str("unit", k).Str("what", "Leaked resource").Msg(Location())
		}
	}
}

// +----------+
// | MultiErr |
// +----------+

type MultiErr struct {
	err   error
	mutex sync.Mutex
}

func NewMultiErr(initial error) *MultiErr {
	return &MultiErr{
		err:   initial,
		mutex: sync.Mutex{},
	}
}

func (m *MultiErr) Append(right error) {
	m.mutex.Lock()
	m.err = multierr.Append(m.err, right)
	m.mutex.Unlock()
}

func (m *MultiErr) Err() error {
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
