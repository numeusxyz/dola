package dola_test

import (
	"github.com/numus-digital/dola"
)

func ExampleKeep() {
	keep, _ := dola.NewKeepBuilder().Build()
	keep.Root.Add("verbose", dola.VerboseStrategy{}) //nolint:exhaustivestruct
	keep.Run()
}
