// The toolbelt is a set of helper functions that ease the cross usage of strategies.
package dola

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
)

var (
	ErrNeedBalancesStrategy = errors.New("Keep should be configured with balances support")
	ErrCast                 = errors.New("casting failed")
)

func CurrencyBalance(k *Keep, exchangeName, currencyCode string, accountIndex int) (account.Balance, error) {
	st, err := k.Root.Get("balances")
	if errors.Is(err, ErrStrategyNotFound) {
		var empty account.Balance

		return empty, ErrNeedBalancesStrategy
	}

	balances, ok := st.(*BalancesStrategy)
	if !ok {
		var empty account.Balance

		return empty, ErrCast
	}

	return balances.Currency(exchangeName, currencyCode, accountIndex)
}
