package dola

import (
	"errors"
	"sync"

	"go.uber.org/multierr"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// +----------+
// | Strategy |
// +----------+

type Strategy interface {
	Init(e exchange.IBotExchange) error
	OnFunding(e exchange.IBotExchange, x stream.FundingData) error
	OnPrice(e exchange.IBotExchange, x ticker.Price) error
	OnKline(e exchange.IBotExchange, x stream.KlineData) error
	OnOrderBook(e exchange.IBotExchange, x orderbook.Base) error
	OnOrder(e exchange.IBotExchange, x order.Detail) error
	OnModify(e exchange.IBotExchange, x order.Modify) error
	OnBalanceChange(e exchange.IBotExchange, x account.Change) error
	Deinit(e exchange.IBotExchange) error
}

// +---------+
// | Manager |
// +---------+

type Manager struct {
	strategies sync.Map
}

func (m *Manager) Add(name string, s Strategy) error {
	_, loaded := m.strategies.LoadOrStore(name, s)
	if loaded {
		return errors.New("strategy already stored")
	}
	return nil
}

func (m *Manager) Delete(name string, e exchange.IBotExchange) error {
	x, ok := m.strategies.LoadAndDelete(name)
	if !ok {
		return errors.New("strategy not found")
	}
	s := x.(Strategy)
	return s.Deinit(e)
}

// +------------------------------+
// | Manager + Strategy interface |
// +------------------------------+

func (m *Manager) each(f func(Strategy) error) error {
	var err error
	m.strategies.Range(func(key, value interface{}) bool {
		s := value.(Strategy)
		err = multierr.Append(err, f(s))
		return true
	})
	return err
}

func (m *Manager) Init(e exchange.IBotExchange) error {
	return m.each(func(s Strategy) error { return s.Init(e) })
}

func (m *Manager) OnFunding(e exchange.IBotExchange, x stream.FundingData) error {
	return m.each(func(s Strategy) error { return s.OnFunding(e, x) })
}

func (m *Manager) OnPrice(e exchange.IBotExchange, x ticker.Price) error {
	return m.each(func(s Strategy) error { return s.OnPrice(e, x) })
}

func (m *Manager) OnKline(e exchange.IBotExchange, x stream.KlineData) error {
	return m.each(func(s Strategy) error { return s.OnKline(e, x) })
}

func (m *Manager) OnOrderBook(e exchange.IBotExchange, x orderbook.Base) error {
	return m.each(func(s Strategy) error { return s.OnOrderBook(e, x) })
}

func (m *Manager) OnOrder(e exchange.IBotExchange, x order.Detail) error {
	return m.each(func(s Strategy) error { return s.OnOrder(e, x) })
}

func (m *Manager) OnModify(e exchange.IBotExchange, x order.Modify) error {
	return m.each(func(s Strategy) error { return s.OnModify(e, x) })
}

func (m *Manager) OnBalanceChange(e exchange.IBotExchange, x account.Change) error {
	return m.each(func(s Strategy) error { return s.OnBalanceChange(e, x) })
}

func (m *Manager) Deinit(e exchange.IBotExchange) error {
	return m.each(func(s Strategy) error { return s.Deinit(e) })
}
