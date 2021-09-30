// Holdings contains related functions / types
package dola

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Balance is a sub type to store currency name and individual totals.
type CurrencyBalance struct {
	Currency   currency.Code
	TotalValue float64
	Hold       float64
}

// SubAccount defines a singular account type with asocciated currency balances.
type SubAccount struct {
	ID       string
	Balances map[asset.Item]map[currency.Code]CurrencyBalance
}

// Holdings maps account ids to SubAccounts.
type ExchangeHoldings struct {
	Accounts map[string]SubAccount
}

func NewExchangeHoldings() *ExchangeHoldings {
	return &ExchangeHoldings{
		Accounts: make(map[string]SubAccount),
	}
}

func (h *ExchangeHoldings) CurrencyBalance(accountID string,
	asset asset.Item,
	code currency.Code) (CurrencyBalance, error) {
	account, ok := h.Accounts[accountID]
	if !ok {
		var empty CurrencyBalance

		return empty, ErrAccountNotFound
	}

	currency, ok := account.Balances[asset][code]
	if !ok {
		var empty CurrencyBalance

		return empty, ErrCurrencyNotFound
	}

	return currency, nil
}
