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
	Init(k *Keep, e exchange.IBotExchange) error
	OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error
	OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error
	OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error
	OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error
	OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error
	OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error
	OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error
	OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error
	Deinit(k *Keep, e exchange.IBotExchange) error
}
