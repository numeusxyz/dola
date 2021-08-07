package dola

import (
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type Subscriber func(ticker.Price) bool

type Multiplexer struct {
	xs  sync.Map
	key int32
}

func (s *Multiplexer) Add(f Subscriber) int32 {
	key := atomic.AddInt32(&s.key, 1)
	s.xs.Store(key, f)
	return key
}

func (s *Multiplexer) Remove(key int32) {
	s.xs.Delete(key)
}

func (s *Multiplexer) Apply(p ticker.Price) {
	remove := make([]int32, 0, 4)
	s.xs.Range(func(key, value interface{}) bool {
		f := value.(Subscriber)
		if !f(p) {
			remove = append(remove, key.(int32))
		}
		return true
	})
	for _, key := range remove {
		s.Remove(key)
	}
}
