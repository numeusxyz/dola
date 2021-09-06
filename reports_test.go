package dola_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/numus-digital/dola"
)

func TestHistorian_State(t *testing.T) {
	t.Parallel()

	u := dola.NewHistorian("", 3, nil)
	f := func(xs ...float64) {
		t.Helper()

		if diff := cmp.Diff(xs, u.Floats()); diff != "" {
			t.Error(diff)
		}
	}

	u.Push(2.0)
	f(2)

	u.Push(4.0)
	f(2, 4)

	u.Push(8.0)
	f(2, 4, 8)

	u.Push(16.0)
	f(4, 8, 16)
}

func TestHistorian_Floats(t *testing.T) {
	t.Parallel()

	u := dola.NewHistorian("", 5, nil)
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
