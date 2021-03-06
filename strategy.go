package dola

import (
	"context"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
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
	Init(ctx context.Context, k *Keep, e exchange.IBotExchange) error
	OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error
	OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error
	OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error
	OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error
	OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error
	OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error
	OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error
	OnTrade(k *Keep, e exchange.IBotExchange, x []trade.Data) error
	OnFill(k *Keep, e exchange.IBotExchange, x []fill.Data) error
	OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error
	Deinit(k *Keep, e exchange.IBotExchange) error
}
