package dola

import (
	"fmt"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type VerboseStrategy struct {
	SilencePrice     bool
	SilenceKline     bool
	SilenceOrderBook bool
}

func (v VerboseStrategy) Init() error {
	fmt.Println("VerboseStrategy.Init()")
	return nil
}

func (v VerboseStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	fmt.Printf("VerboseStrategy.OnFunding(): e=%s x=%v\n", e.GetName(), x)
	return nil
}

func (v VerboseStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	if !v.SilencePrice {
		fmt.Printf("VerboseStrategy.OnPrice(): e=%s, x=%v\n", e.GetName(), x)
	}
	return nil
}

func (v VerboseStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	if !v.SilenceKline {
		fmt.Printf("VerboseStrategy.OnKline(): e=%s, x=%v\n", e.GetName(), x)
	}
	return nil
}

func (v VerboseStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	if !v.SilenceOrderBook {
		fmt.Printf(
			"VerboseStrategy.OnOrderBook(): e=%s, x.Asks=%d x.Bids=%d ask=%f bid=%f\n",
			e.GetName(), len(x.Asks), len(x.Bids), x.Asks[0].Price, x.Bids[0].Price,
		)
	}
	return nil
}

func (v VerboseStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	fmt.Printf("VerboseStrategy.OnOrder(): e=%s, x=%v\n", e.GetName(), x)
	return nil
}

func (v VerboseStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	fmt.Printf("VerboseStrategy.OnModify(): e=%s, x=%v\n", e.GetName(), x)
	return nil
}

func (v VerboseStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	fmt.Printf("VerboseStrategy.OnBalanceChange(): e=%s, x=%v\n", e.GetName(), x)
	return nil
}

func (v VerboseStrategy) OnOrderPlace(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	fmt.Printf("VerboseStrategy.OnOrderPlace(): e=%s, x=%v\n", e.GetName(), x)
	return nil
}

func (v VerboseStrategy) OnOrderFill(k *Keep, e exchange.IBotExchange, x OrderFill) error {
	fmt.Printf("VerboseStrategy.OnOrderFill(): e=%s, x=%v\n", e.GetName(), x)
	return nil
}

func (v VerboseStrategy) Deinit() error {
	fmt.Println("VerboseStrategy.OnDeinit()")
	return nil
}
