package dola

import (
	"sync"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type TickerStrategy struct {
	Interval time.Duration
	TickFunc func(k *Keep, e exchange.IBotExchange)
	tickers  sync.Map
}

func (s *TickerStrategy) Init(k *Keep, e exchange.IBotExchange) error {
	ticker := *time.NewTicker(s.Interval)

	if s.TickFunc != nil {
		go func() {
			CheckerPush()

			defer CheckerPop()

			// Call now initially.
			s.TickFunc(k, e)

			for range ticker.C {
				s.TickFunc(k, e)
			}
		}()
	}

	_, loaded := s.tickers.LoadOrStore(e.GetName(), ticker)
	if loaded {
		panic("one exchange can have just one ticker")
	}

	return nil
}

func (s *TickerStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return nil
}

func (s *TickerStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	return nil
}

func (s *TickerStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return nil
}

func (s *TickerStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return nil
}

func (s *TickerStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return nil
}

func (s *TickerStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return nil
}

func (s *TickerStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return nil
}

func (s *TickerStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (s *TickerStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	pointer, loaded := s.tickers.LoadAndDelete(e.GetName())
	if !loaded {
		panic("exchange has no registered ticker")
	}

	ticker, ok := pointer.(time.Ticker)
	if !ok {
		panic("want time.Ticker")
	}

	ticker.Stop()

	return nil
}
