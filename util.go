package dola

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

// Location returns the name of the caller function.
func Location() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "?"
	}

	fn := runtime.FuncForPC(pc)
	xs := strings.SplitAfterN(fn.Name(), "/", 3) // nolint: gomnd

	return xs[len(xs)-1]
}

// Location2 returns the name of the caller of the caller function.
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

// Checker is a simple tool to check if everything initialized is subsequently
// deinitialized.  Works from simple open/close calls to gourintes.
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
	What(log.Debug(), "checking resources...")
	time.Sleep(1 * time.Second)

	defaultResourceChecker.m.Lock()
	defer defaultResourceChecker.m.Unlock()

	for k, v := range defaultResourceChecker.resources {
		if v != 0 {
			What(log.Warn().Int("counter", v).Str("unit", k), "leaked resource")
		}
	}
}

// +----------------+
// | ErrorWaitGroup |
// +----------------+

type ErrorWaitGroup struct {
	err   error
	group sync.WaitGroup
	mutex sync.Mutex
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

// +---------------+
// | Miscellaneous |
// +---------------+

// ConfigFile checks whether a file with the given path exists.  If it doesnt, it falls
// back to: (1) $DOLA_CONFIG, (2) ~/.dola.config.json or (3) returns an empty string.
func ConfigFile(inp string) string {
	if inp != "" {
		path := ExpandUser(inp)
		if FileExists(path) {
			return path
		}
	}

	if env := os.Getenv("DOLA_CONFIG"); env != "" {
		path := ExpandUser(env)
		if FileExists(path) {
			return path
		}
	}

	if path := ExpandUser("~/.dola/config.json"); FileExists(path) {
		return path
	}

	return ""
}

// ExpandUser returns the argument with an initial ~ replaced by user's home directory.
func ExpandUser(path string) string {
	// Maybe this won't work on Windows, but do we care?
	return os.ExpandEnv(strings.Replace(path, "~", "$HOME", 1))
}

func FileExists(path string) bool {
	_, err := os.Stat(path)

	return !os.IsNotExist(err)
}

// RandomOrderID uses code and ideas from:
// https://stackoverflow.com/questions/32349807 and
// https://stackoverflow.com/questions/13378815 .
//
// Length of produced client order ID is encoded in the code.  See `seed`.
func RandomOrderID(prefix string) (string, error) {
	const seed = 24
	xs := make([]byte, seed)

	if _, err := rand.Read(xs); err != nil {
		return "", err
	}

	ys := base64.URLEncoding.EncodeToString(xs)
	offset := len(prefix)
	id := fmt.Sprintf("%s%s", prefix, ys[offset:])

	return id, nil
}
