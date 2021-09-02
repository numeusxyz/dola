package dola

import (
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type Reporter struct {
	RefreshTime time.Duration
}

func (r *Reporter) Init(k *Keep, e exchange.IBotExchange) error {
	return nil
}

func (r *Reporter) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return nil
}

func (r *Reporter) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	return nil
}

func (r *Reporter) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return nil
}

func (r *Reporter) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return nil
}

func (r *Reporter) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return nil
}

func (r *Reporter) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return nil
}

func (r *Reporter) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return nil
}

func (r *Reporter) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (r *Reporter) Deinit(k *Keep, e exchange.IBotExchange) error {
	return nil
}
