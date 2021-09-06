package dola_test

import (
	"testing"

	"github.com/numus-digital/dola"
)

//nolint:funlen
func TestCircularArray(t *testing.T) {
	t.Parallel()

	a := dola.NewCircularArray(3)

	offset := func(offset int) {
		if a.Offset != offset {
			t.Errorf("have %d, want %d", a.Offset, offset)
		}
	}

	last := func(last int) {
		if a.LastIndex() != last {
			t.Errorf("have %d, want %d", a.LastIndex(), last)
		}
	}

	values := func(xs ...int) {
		if a.Len() > len(xs) {
			t.Error()
		}

		for i, x := range xs {
			value, ok := a.At(i).(int)
			if !ok {
				t.Error()
			}

			if value != x {
				t.Errorf("have %d, want %d, xs=%v", value, x, xs)
			}
		}
	}

	offset(0)
	values()

	a.Push(0)
	offset(0)
	last(0)
	values(0)

	a.Push(1)
	offset(0)
	last(1)
	values(0, 1)

	a.Push(2)
	offset(0)
	last(2)
	values(0, 1, 2)

	a.Push(3) // [3, 1*, 2]
	offset(1)
	last(0)
	values(1, 2, 3)

	a.Push(4) // [3, 4, 2*]
	offset(2)
	last(1)
	values(2, 3, 4)

	a.Push(5) // [3*, 4, 5]
	offset(0)
	last(2)
	values(3, 4, 5)

	a.Push(6) // [6, 4*, 5]
	offset(1)
	last(0)
	values(4, 5, 6)

	a.Push(7) // [6, 7, 5*]
	offset(2)
	last(1)
	values(5, 6, 7)

	a.Push(8) // [6*, 7, 8]
	offset(0)
	last(2)
	values(6, 7, 8)

	a.Push(9) // [9, 7*, 8]
	offset(1)
	last(0)
	values(7, 8, 9)

	a.Push(10) // [9, 10, 8*]
	offset(2)
	last(1)
	values(8, 9, 10)

	a.Push(11) // [9*, 10, 11]
	offset(0)
	last(2)
	values(9, 10, 11)
}

func TestCircularArray_Index(t *testing.T) {
	t.Parallel()

	xs := dola.NewCircularArray(3)
	f := func(x, want int) {
		t.Helper()

		have := xs.Index(x)
		if have != want {
			t.Errorf("have=%d, want=%d, x=%d, offset=%d", have, want, x, xs.Offset)
		}
	}

	xs.Offset = 0

	f(0, 0)
	f(1, 1)
	f(2, 2)

	xs.Offset = 1

	f(0, 1)
	f(1, 2)
	f(2, 0)

	xs.Offset = 2

	f(0, 2)
	f(1, 0)
	f(2, 1)
}
