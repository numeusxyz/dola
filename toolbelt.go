// The toolbelt is a set of helper functions that eases strategies and cross usage.
package dola

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var ErrNeedBalancesStrategy = errors.New("Keep should be configured with balances support")

func CurrencyBalance(k *Keep, exchangeName, currencyCode string, accountIndex int) (account.Balance, error) {
	st, err := k.Root.Get("balances")
	if errors.Is(err, ErrStrategyNotFound) {
		var empty account.Balance

		return empty, ErrNeedBalancesStrategy
	}

	balances, ok := st.(*BalancesStrategy)
	if !ok {
		panic("casting failed")
	}

	return balances.Currency(exchangeName, currencyCode, accountIndex)
}

func ModifyOrder(k *Keep, e exchange.IBotExchange, mod order.Modify) error {
	// First, try to use the native exchange functionality.
	_, err := k.Modify(e, mod)
	if err == nil {
		return nil
	}

	// There is an error.  If it's different than
	// ErrFunctionNotSupported, then modification is (apparently)
	// supported, but failed.
	if !errors.Is(err, common.ErrFunctionNotSupported) {
		return err
	}

	// Modification is not supported.  Fall back to cancellation
	// and submission.

	// First, cancel the order.
	cancel := ModifyToCancel(mod)
	if err := k.CancelOrder(e, cancel); err != nil {
		return err
	}

	// Prepare submission struct.
	submit := ModifyToSubmit(mod)

	// Second, check if there is a UserData associated with that
	// order.  If there is, resubmit the order with the same
	// UserData.
	value, loaded := k.GetOrderValue(e.GetName(), mod.ID)
	if loaded {
		_, err := k.SubmitOrderUD(e, submit, value.UserData)

		return err
	}

	// If there's not, just submit the order.
	_, err = k.SubmitOrder(e, submit)

	return err
}

// Ticker casts a void* to ticker.Price.
func Ticker(p interface{}) ticker.Price {
	x, ok := p.(ticker.Price)
	if !ok {
		panic("")
	}

	return x
}

// +---------------+
// | Conversations |
// +---------------+

func ModifyToCancel(mod order.Modify) order.Cancel {
	var cancel order.Cancel
	// These four are what the GCT engine uses to match an order
	// uniquely.
	cancel.Exchange = mod.Exchange
	cancel.ID = mod.ID
	cancel.AssetType = mod.AssetType
	cancel.Pair = mod.Pair

	return cancel
}

func ModifyToSubmit(mod order.Modify) order.Submit {
	sub := order.Submit{
		ImmediateOrCancel: mod.ImmediateOrCancel,
		HiddenOrder:       mod.HiddenOrder,
		FillOrKill:        mod.FillOrKill,
		PostOnly:          mod.PostOnly,
		ReduceOnly:        false, // Missing.
		Leverage:          mod.Leverage,
		Price:             mod.Price,
		Amount:            mod.Amount,
		StopPrice:         0, // Missing.
		LimitPriceUpper:   mod.LimitPriceUpper,
		LimitPriceLower:   mod.LimitPriceLower,
		TriggerPrice:      mod.TriggerPrice,
		TargetAmount:      mod.TargetAmount,
		ExecutedAmount:    mod.ExecutedAmount,
		RemainingAmount:   mod.RemainingAmount,
		Fee:               mod.Fee,
		Exchange:          mod.Exchange,
		InternalOrderID:   mod.InternalOrderID,
		ID:                mod.ID,
		AccountID:         mod.AccountID,
		ClientID:          mod.ClientID,
		ClientOrderID:     mod.ClientOrderID,
		WalletAddress:     mod.WalletAddress,
		Offset:            "", // Missing.
		Type:              mod.Type,
		Side:              mod.Side,
		Status:            mod.Status,
		AssetType:         mod.AssetType,
		Date:              mod.Date,
		LastUpdated:       mod.LastUpdated,
		Pair:              mod.Pair,
		Trades:            mod.Trades,
	}

	return sub
}
