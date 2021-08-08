package dola

import (
	"errors"
	"fmt"
	"sync"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"go.uber.org/multierr"
)

var (
	ErrStrategyNotFound      = errors.New("strategy not found")
	ErrStrategyAlreadyExists = errors.New("strategy already exists")
)

// +----------+
// | Strategy |
// +----------+

type Trade struct {
	Timestamp     time.Time
	BaseCurrency  string
	QuoteCurrency string
	OrderID       string
	AveragePrice  float64
	Quantity      float64
	Fee           float64
	FeeCurrency   string
}

type Strategy interface {
	Init() error
	OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error
	OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error
	OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error
	OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error
	OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error
	OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error
	OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error
	Deinit() error
}

// +---------+
// | Manager |
// +---------+

type RootStrategy struct {
	strategies sync.Map
}

func (m *RootStrategy) Add(name string, s Strategy) error {
	if err := s.Init(); err != nil {
		return fmt.Errorf("Strategy.Init failed: %w", err)
	}

	if _, loaded := m.strategies.LoadOrStore(name, s); loaded {
		return ErrStrategyAlreadyExists
	}

	return nil
}

func (m *RootStrategy) Delete(name string) error {
	x, ok := m.strategies.LoadAndDelete(name)

	if !ok {
		return ErrStrategyNotFound
	}
	s := x.(Strategy)

	return s.Deinit()
}

// +------------------------------+
// | Manager + Strategy interface |
// +------------------------------+

func (m *RootStrategy) each(f func(Strategy) error) error {
	var err error

	m.strategies.Range(func(key, value interface{}) bool {
		s := value.(Strategy)
		err = multierr.Append(err, f(s))

		return true
	})

	return err
}

func (m *RootStrategy) Init() error {
	return m.each(func(s Strategy) error { return s.Init() })
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

func (m *RootStrategy) Deinit() error {
	return m.each(func(s Strategy) error { return s.Deinit() })
}
