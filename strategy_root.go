package dola

import (
	"errors"
	"sync"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"go.uber.org/multierr"
)

// +---------+
// | Manager |
// +---------+

var (
	ErrStrategyNotFound = errors.New("strategy not found")
	ErrNotStrategy      = errors.New("given object is not a strategy")
)

type RootStrategy struct {
	strategies sync.Map
}

func NewRootStrategy() RootStrategy {
	return RootStrategy{
		strategies: sync.Map{},
	}
}

func (m *RootStrategy) Add(name string, s Strategy) {
	m.strategies.Store(name, s)
}

func (m *RootStrategy) Delete(name string) (Strategy, error) {
	x, ok := m.strategies.LoadAndDelete(name)
	if !ok {
		return nil, ErrStrategyNotFound
	}

	return x.(Strategy), nil
}

func (m *RootStrategy) Get(name string) (Strategy, error) {
	x, ok := m.strategies.Load(name)
	if !ok {
		return nil, ErrStrategyNotFound
	}

	return x.(Strategy), nil
}

// +----------+
// | Strategy |
// +----------+

func (m *RootStrategy) each(f func(Strategy) error) error {
	var err error

	m.strategies.Range(func(key, value interface{}) bool {
		s, ok := value.(Strategy)
		if !ok {
			err = multierr.Append(err, ErrNotStrategy)
		} else {
			err = multierr.Append(err, f(s))
		}

		return true
	})

	return err
}

func (m *RootStrategy) Init(k *Keep, e exchange.IBotExchange) error {
	return m.each(func(s Strategy) error { return s.Init(k, e) })
}

func (m *RootStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return m.each(func(s Strategy) error { return s.OnFunding(k, e, x) })
}

func (m *RootStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	return m.each(func(s Strategy) error { return s.OnPrice(k, e, x) })
}

func (m *RootStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return m.each(func(s Strategy) error { return s.OnKline(k, e, x) })
}

func (m *RootStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return m.each(func(s Strategy) error { return s.OnOrderBook(k, e, x) })
}

func (m *RootStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return m.each(func(s Strategy) error { return s.OnOrder(k, e, x) })
}

func (m *RootStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return m.each(func(s Strategy) error { return s.OnModify(k, e, x) })
}

func (m *RootStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return m.each(func(s Strategy) error { return s.OnBalanceChange(k, e, x) })
}

func (m *RootStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return m.each(func(s Strategy) error { return s.OnUnrecognized(k, e, x) })
}

func (m *RootStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	return m.each(func(s Strategy) error { return s.Deinit(k, e) })
}
