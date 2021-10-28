package dola

import (
	"context"

	"github.com/rs/zerolog/log"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

type VerboseStrategy struct {
	SilencePrice     bool
	SilenceKline     bool
	SilenceOrderBook bool
	SilenceOrder     bool
}

func (v VerboseStrategy) Init(ctx context.Context, k *Keep, e exchange.IBotExchange) error {
	Msg(log.Info().Str("e", e.GetName()))

	return nil
}

func (v VerboseStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	Msg(log.Info().Str("e", e.GetName()).Interface("x", x))

	return nil
}

func (v VerboseStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	if !v.SilencePrice {
		Msg(log.Info().Str("e", e.GetName()).Interface("x", x))
	}

	return nil
}

func (v VerboseStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	if !v.SilenceKline {
		Msg(log.Info().Str("e", e.GetName()).Interface("x", x))
	}

	return nil
}

func (v VerboseStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	if !v.SilenceOrderBook {
		askPrice := 0.0
		askAmount := 0.0

		if len(x.Asks) > 0 {
			askPrice = x.Asks[0].Price
			askAmount = x.Asks[0].Amount
		}

		bidPrice := 0.0
		bidAmount := 0.0

		if len(x.Bids) > 0 {
			bidPrice = x.Bids[0].Price
			bidAmount = x.Bids[0].Amount
		}

		Msg(log.Info().
			Str("asset", string(x.Asset)).
			Str("pair", x.Pair.String()).
			Float64("ask_price", askPrice).
			Float64("bid_price", bidPrice).
			Float64("ask_size", askAmount).
			Float64("bid_size", bidAmount).
			Int("len(asks)", len(x.Asks)).
			Int("len(bids)", len(x.Bids)).
			Str("e", e.GetName()))
	}

	return nil
}

func (v VerboseStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	if !v.SilenceOrder {
		Msg(log.Info().Str("e", e.GetName()).Interface("x", x))
	}

	return nil
}

func (v VerboseStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	Msg(log.Info().Str("e", e.GetName()).Interface("x", x))

	return nil
}

func (v VerboseStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	Msg(log.Info().Str("e", e.GetName()).Interface("x", x))

	return nil
}

func (v VerboseStrategy) OnTrade(k *Keep, e exchange.IBotExchange, x []trade.Data) error {
	Msg(log.Info().Str("e", e.GetName()).Interface("x", x))

	return nil
}

func (v VerboseStrategy) OnFill(k *Keep, e exchange.IBotExchange, x []fill.Data) error {
	Msg(log.Info().Str("e", e.GetName()).Interface("x", x))

	return nil
}

func (v VerboseStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	Msg(log.Info().Str("e", e.GetName()).Interface("x", x))

	return nil
}

func (v VerboseStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	Msg(log.Info().Str("e", e.GetName()))

	return nil
}
