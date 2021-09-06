package dola

import (
	"fmt"
	"reflect"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// +--------------+
// | ReporterUnit |
// +--------------+

type ReporterUnit struct {
	Key   string
	State []interface{}
}

func NewReporterUnit(key string, n int) ReporterUnit {
	return ReporterUnit{
		Key:   key,
		State: make([]interface{}, 0, n),
	}
}

func (u *ReporterUnit) Push(x interface{}) {
	n := len(u.State)

	// Make sure every Push() is called with the same argument type.
	if n > 0 {
		if have, want := reflect.TypeOf(x), reflect.TypeOf(u.State[0]); have != want {
			panic(fmt.Sprintf("wrong type: have %v, want %v", have, want))
		}
	}

	if n == cap(u.State) {
		// Make room for this element by scratching off the first one.
		//
		// TODO: This causes a memory leak!
		u.State = u.State[1:]
	}

	fmt.Printf("BEFORE: len=%d cap=%d\n", len(u.State), cap(u.State))
	u.State = append(u.State, x)
	fmt.Printf("AFTER: len=%d cap=%d\n", len(u.State), cap(u.State))

}

// Floats returns the State array, but casted to []float64.
func (u *ReporterUnit) Floats() []float64 {
	ys := make([]float64, len(u.State))
	for i, x := range u.State {
		y, ok := x.(float64)
		if !ok {
			panic(fmt.Sprintf("illegal type: %T", x))
		}
		ys[i] = y
	}

	return ys
}

func (u *ReporterUnit) Last() interface{} {
	//
	return nil
}

// +----------+
// | Reporter |
// +----------+

type Reporter struct {
	RefreshTime  time.Duration
	onPriceUnits map[string]ReporterUnit
	onOrderUnits map[string]ReporterUnit
}

func (r *Reporter) Init(k *Keep, e exchange.IBotExchange) error {
	return nil
}

func (r *Reporter) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return nil
}

func (r *Reporter) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	// r.onPriceUnits[e.GetName()].State

	return nil
}

func (r *Reporter) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return nil
}

func (r *Reporter) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return nil
}

func (r *Reporter) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return nil
}

func (r *Reporter) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return nil
}

func (r *Reporter) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return nil
}

func (r *Reporter) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (r *Reporter) Deinit(k *Keep, e exchange.IBotExchange) error {
	return nil
}
