package dola

type CircularArray struct {
	Offset int
	xs     []interface{}
}

func NewCircularArray(n int) CircularArray {
	xs := make([]interface{}, 0, n)
	if cap(xs) != n {
		panic("")
	}

	return CircularArray{
		Offset: 0,
		xs:     xs,
	}
}

func (a *CircularArray) Len() int {
	return len(a.xs)
}

func (a *CircularArray) At(index int) interface{} {
	mapped := a.Index(index)

	return a.xs[mapped]
}

// Index maps an external 0-based index to the corresponding internal index.
func (a *CircularArray) Index(i int) int {
	return (i + a.Offset) % cap(a.xs)
}

func (a *CircularArray) LastIndex() int {
	return a.Index(len(a.xs) - 1)
}

func (a *CircularArray) Push(x interface{}) {
	if len(a.xs) < cap(a.xs) {
		a.xs = append(a.xs, x)
	} else {
		a.Offset = (a.Offset + 1) % cap(a.xs)
		last := a.LastIndex()
		a.xs[last] = x
	}
}
