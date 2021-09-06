package dola_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/numus-digital/dola"
)

func TestReporterUnit_State(t *testing.T) {
	t.Parallel()

	u := dola.NewReporterUnit("", 3)
	f := func(xs ...int) {
		t.Helper()

		if have, want := len(u.State), len(xs); have != want {
			t.Errorf("wrong length: have %d, want %d, xs=%v", have, want, xs)
		}

		for i, want := range xs {
			have, ok := u.State[i].(int)
			if !ok {
				t.Errorf("wrong element type: have %T, want int, xs=%v", u.State[i], xs)
			}

			if have != want {
				t.Errorf("wrong element value: have %d, want %d, xs=%v", have, want, xs)
			}
		}
	}

	f()

	u.Push(10)
	f(10)

	u.Push(20)
	f(10, 20)

	u.Push(30)
	f(10, 20, 30)

	u.Push(40)
	f(20, 30, 40)
}

func TestReporterUnit_Floats(t *testing.T) {
	t.Parallel()

	u := dola.NewReporterUnit("", 5)
	u.Push(2.0)
	u.Push(4.0)
	u.Push(8.0)
	u.Push(16.0)
	u.Push(32.0)

	xs := []float64{2, 4, 8, 16, 32}
	if diff := cmp.Diff(u.Floats(), xs); diff != "" {
		t.Errorf(diff)
	}
}
