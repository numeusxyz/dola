package dola

import (
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Location returns the name of the parent calling function.
func Location() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "?"
	}
	fn := runtime.FuncForPC(pc)
	return fn.Name()
}

// Location2 returns the name of the grandparent calling function.
func Location2() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return "?"
	}
	fn := runtime.FuncForPC(pc)
	return fn.Name()
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

var defaultResourceChecker = resourceChecker{resources: make(map[string]int)}

func CheckerPush(xs ...string) {
	var name string
	switch len(xs) {
	case 0:
		name = Location2()
	case 1:
		name = xs[0]
	default:
		panic("invalid arugment")
	}
	defaultResourceChecker.m.Lock()
	defer defaultResourceChecker.m.Unlock()
	defaultResourceChecker.resources[name] += 1
}

func CheckerPop(xs ...string) {
	var name string
	switch len(xs) {
	case 0:
		name = Location2()
	case 1:
		name = xs[0]
	default:
		panic("invalid arugment")
	}
	defaultResourceChecker.m.Lock()
	defer defaultResourceChecker.m.Unlock()
	defaultResourceChecker.resources[name] -= 1
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
	err = pprof.StartCPUProfile(f)
	if err != nil {
		panic(err)
	}
	return p
}

func (p Profiler) Stop() {
	pprof.StopCPUProfile()
}
