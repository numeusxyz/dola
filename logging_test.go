package dola_test

import (
	"testing"
	"time"

	"github.com/numeusxyz/dola"
)

func TestLogState(t *testing.T) {
	t.Parallel()

	state := dola.NewLogState(100 * time.Millisecond)

	// Make sure repeated calls to Awaken() report the same state.
	for i := 0; i < 3; i++ {
		if state.Awaken() {
			t.Error("have awaken, want dormant")
		}
	}

	// Put into awaken state.
	state.WakeUp()

	// Make sure for the predefined duration state stays awaken.
	for i := 0; i < 10; i++ {
		if !state.Awaken() {
			t.Error("have dormant, want awaken")
		}

		time.Sleep(10 * time.Millisecond)
	}

	// After time expires, make sure state is dormant again.
	for i := 0; i < 3; i++ {
		if state.Awaken() {
			t.Error("have awaken, want dormant")
		}
	}
}
