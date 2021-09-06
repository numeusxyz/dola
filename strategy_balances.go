package dola

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// +------------------+
// | BalancesStrategy |
// +------------------+

var (
	ErrAccountIndexOutOfRange = errors.New("no account with this index exists")
	ErrCurrencyNotFound       = errors.New("currency not found in holdings")
	ErrHoldingsNotFound       = errors.New("holdings not found for exchange")
)

type BalancesStrategy struct {
	balances sync.Map
	ticker   TickerStrategy
}

func NewBalancesStrategy(refreshRate time.Duration) Strategy {
	b := &BalancesStrategy{
		balances: sync.Map{},
		ticker: TickerStrategy{
			Interval: refreshRate,
			TickFunc: nil,
			tickers:  sync.Map{},
		},
	}
	b.ticker.TickFunc = b.tick

	return b
}

func (b *BalancesStrategy) Store(holdings account.Holdings) {
	b.balances.Store(holdings.Exchange, holdings)
}

func (b *BalancesStrategy) Load(exchangeName string) (holdings account.Holdings, loaded bool) {
	pointer, loaded := b.balances.Load(exchangeName)

	if loaded {
		var ok bool
		holdings, ok = pointer.(account.Holdings)

		if !ok {
			panic(fmt.Sprintf("have %T, want account.Holdings", pointer))
		}
	}

	return
}

func (b *BalancesStrategy) Currency(exchangeName string, code string, accountIndex int) (account.Balance, error) {
	holdings, loaded := b.Load(exchangeName)
	if !loaded {
		var empty account.Balance

		return empty, ErrHoldingsNotFound
	}

	if len(holdings.Accounts) < accountIndex {
		var empty account.Balance

		return empty, ErrAccountIndexOutOfRange
	}

	sub := holdings.Accounts[accountIndex]
	for _, balance := range sub.Currencies {
		if balance.CurrencyName.String() == code {
			return balance, nil
		}
	}

	var empty account.Balance

	return empty, ErrCurrencyNotFound
}

func (b *BalancesStrategy) tick(k *Keep, e exchange.IBotExchange) {
	holdings, err := e.UpdateAccountInfo(asset.Spot)
	if err != nil {
		What(log.Error().Str("exchange", e.GetName()).Err(err), "")
	}

	b.Store(holdings)
}

// +--------------------+
// | Strategy interface |
// +--------------------+

func (b *BalancesStrategy) Init(k *Keep, e exchange.IBotExchange) error {
	return b.ticker.Init(k, e)
}

func (b *BalancesStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return nil
}

func (b *BalancesStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	return nil
}

func (b *BalancesStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return nil
}

func (b *BalancesStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return nil
}

func (b *BalancesStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return nil
}

func (b *BalancesStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return nil
}

func (b *BalancesStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return nil
}

func (b *BalancesStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (b *BalancesStrategy) OnEvent(k *Keep, x interface{}) error {
	return nil
}

func (b *BalancesStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	return b.ticker.Init(k, e)
}
