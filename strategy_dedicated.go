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

func (d *DedicatedStrategy) Init(e exchange.IBotExchange) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.Init(e)
	}
	return nil
}

func (d *DedicatedStrategy) OnFunding(e exchange.IBotExchange, x stream.FundingData) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnFunding(e, x)
	}
	return nil
}

func (d *DedicatedStrategy) OnPrice(e exchange.IBotExchange, x ticker.Price) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnPrice(e, x)
	}
	return nil
}

func (d *DedicatedStrategy) OnKline(e exchange.IBotExchange, x stream.KlineData) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnKline(e, x)
	}
	return nil
}

func (d *DedicatedStrategy) OnOrderBook(e exchange.IBotExchange, x orderbook.Base) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnOrderBook(e, x)
	}
	return nil
}

func (d *DedicatedStrategy) OnOrder(e exchange.IBotExchange, x order.Detail) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnOrder(e, x)
	}
	return nil
}

func (d *DedicatedStrategy) OnModify(e exchange.IBotExchange, x order.Modify) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnModify(e, x)
	}
	return nil
}

func (d *DedicatedStrategy) OnBalanceChange(e exchange.IBotExchange, x account.Change) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.OnBalanceChange(e, x)
	}
	return nil
}

func (d *DedicatedStrategy) Deinit(e exchange.IBotExchange) error {
	if e.GetName() == d.Exchange {
		return d.Wrapped.Deinit(e)
	}
	return nil
}
