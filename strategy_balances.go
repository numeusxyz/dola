package dola

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// +------------------+
// | BalancesStrategy |
// +------------------+

var (
	ErrAccountIndexOutOfRange = errors.New("no account with this index exists")
	ErrHoldingsNotFound       = errors.New("holdings not found for exchange")
	ErrAccountNotFound        = errors.New("account not found in holdings")
	ErrAssetNotFound          = errors.New("asset not found in account")
	ErrCurrencyNotFound       = errors.New("currency not found in asset")
)

type BalancesStrategy struct {
	// holdings maps an exchange name to its holdings
	holdings sync.Map
	ticker   TickerStrategy
}

func NewBalancesStrategy(refreshRate time.Duration) Strategy {
	b := &BalancesStrategy{
		holdings: sync.Map{},
		ticker: TickerStrategy{
			Interval: refreshRate,
			TickFunc: nil,
			tickers:  sync.Map{},
		},
	}
	b.ticker.TickFunc = b.tick

	return b
}

func (b *BalancesStrategy) ExchangeHoldings(exchangeName string) (*ExchangeHoldings, error) {
	key := strings.ToLower(exchangeName)

	if ptr, ok := b.holdings.Load(key); ok {
		if h, ok := ptr.(*ExchangeHoldings); ok {
			return h, nil
		}
	}

	return nil, ErrHoldingsNotFound
}

func (b *BalancesStrategy) tick(k *Keep, e exchange.IBotExchange) {
	// create a new holdings struct that we'll fill out and then
	// atomically update
	holdings := NewExchangeHoldings()

	// go through all the asset types, fetch account info for each of them
	// and aggregate them into dola.Holdings
	for _, assetType := range e.GetAssetTypes(true) {
		h, err := e.UpdateAccountInfo(context.Background(), assetType)
		if err != nil {
			Msg(log.Error().Str("exchange", e.GetName()).Err(err))

			continue
		}

		for _, subAccount := range h.Accounts {
			if _, ok := holdings.Accounts[subAccount.ID]; !ok {
				holdings.Accounts[subAccount.ID] = SubAccount{
					ID:       subAccount.ID,
					Balances: make(map[asset.Item]map[currency.Code]CurrencyBalance),
				}
			}

			if _, ok := holdings.Accounts[subAccount.ID].Balances[assetType]; !ok {
				holdings.Accounts[subAccount.ID].Balances[assetType] = make(map[currency.Code]CurrencyBalance)
			}

			for _, currencyBalance := range subAccount.Currencies {
				holdings.Accounts[subAccount.ID].Balances[assetType][currencyBalance.CurrencyName] = CurrencyBalance{
					Currency:   currencyBalance.CurrencyName,
					TotalValue: currencyBalance.TotalValue,
					Hold:       currencyBalance.Hold,
				}
			}
		}
	}

	key := strings.ToLower(e.GetName())
	b.holdings.Store(key, holdings)
}

// +--------------------+
// | Strategy interface |
// +--------------------+

func (b *BalancesStrategy) Init(ctx context.Context, k *Keep, e exchange.IBotExchange) error {
	key := strings.ToLower(e.GetName())
	b.holdings.Store(key, NewExchangeHoldings())

	return b.ticker.Init(ctx, k, e)
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

func (b *BalancesStrategy) OnTrade(k *Keep, e exchange.IBotExchange, x []trade.Data) error {
	return nil
}

func (b *BalancesStrategy) OnFill(k *Keep, e exchange.IBotExchange, x []fill.Data) error {
	return nil
}

func (b *BalancesStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (b *BalancesStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	return b.ticker.Deinit(k, e)
}
