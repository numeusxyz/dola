package dola

import (
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// DedicatedStrategy is a Strategy wrapper that executes wrapped
// methods only when events come from a particular exchange.
type DedicatedStrategy struct {
	Exchange string
	Wrapped  Strategy
}

func (d *DedicatedStrategy) Init(k *Keep, e exchange.IBotExchange) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.Init(k, e)
	}

	return nil
}

func (d *DedicatedStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnFunding(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnPrice(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnKline(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnOrderBook(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnOrder(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnModify(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnBalanceChange(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnUnrecognized(k, e, x)
	}

	return nil
}

func (d *DedicatedStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.Deinit(k, e)
	}

	return nil
}
