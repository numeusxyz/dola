// Package dola provides cryptocurrency trading primitives: exchange
// integrations (via github.com/thrasher-corp/gocryptotrader),
// event-driven strategies, utilities and more.
package dola

import (
	"math/rand"
	"time"
)

// nolint:gochecknoinits
func init() {
	rand.Seed(time.Now().UnixNano())
}
