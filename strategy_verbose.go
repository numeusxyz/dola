package dola

import (
	"github.com/rs/zerolog/log"
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

func (v VerboseStrategy) Init(k *Keep, e exchange.IBotExchange) error {
	log.Info().Str("e", e.GetName()).Msg(Location())

	return nil
}

func (v VerboseStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	log.Info().Str("e", e.GetName()).Interface("x", x).Msg(Location())

	return nil
}

func (v VerboseStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	if !v.SilencePrice {
		log.Info().Str("e", e.GetName()).Interface("x", x).Msg(Location())
	}

	return nil
}

func (v VerboseStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	if !v.SilenceKline {
		log.Info().Str("e", e.GetName()).Interface("x", x).Msg(Location())
	}

	return nil
}

func (v VerboseStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	if !v.SilenceOrderBook {
		ask := 0.0
		if len(x.Asks) > 0 {
			ask = x.Asks[0].Price
		}

		bid := 0.0
		if len(x.Bids) > 0 {
			bid = x.Bids[0].Price
		}

		log.Info().
			Float64("ask", ask).
			Float64("bid", bid).
			Int("len(asks)", len(x.Asks)).
			Int("len(bids", len(x.Bids)).
			Str("e", e.GetName()).
			Msg(Location())
	}

	return nil
}

func (v VerboseStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	log.Info().Str("e", e.GetName()).Interface("x", x).Msg(Location())

	return nil
}

func (v VerboseStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	log.Info().Str("e", e.GetName()).Interface("x", x).Msg(Location())

	return nil
}

func (v VerboseStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	log.Info().Str("e", e.GetName()).Interface("x", x).Msg(Location())

	return nil
}

func (v VerboseStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	log.Info().Str("e", e.GetName()).Msg(Location())

	return nil
}
