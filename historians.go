package dola

import (
	"fmt"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// +-----------+
// | Historian |
// +-----------+

type Array interface {
	At(index int) interface{}
	Len() int
}

type Historian struct {
	Key   string
	F     func(state Array)
	state Array
}

func NewHistorian(key string, n int, f func(state Array)) Historian {
	state := NewCircularArray(n)

	return Historian{
		Key:   key,
		F:     f,
		state: &state,
	}
}

func (u *Historian) Push(x interface{}) {
	u.state.(*CircularArray).Push(x)
}

// Floats returns the State array, but casted to []float64.
func (u *Historian) Floats() []float64 {
	ys := make([]float64, u.state.Len())

	for i := 0; i < u.state.Len(); i++ {
		x := u.state.At(i)

		if y, ok := x.(float64); !ok {
			panic(fmt.Sprintf("illegal type: %T", x))
		} else {
			ys[i] = y
		}
	}

	return ys
}

func (u *Historian) Last() interface{} {
	index := u.state.Len() - 1

	return u.state.At(index)
}

// +-------------------+
// | HistorianStrategy |
// +-------------------+

type HistorianStrategy struct {
	Interval     time.Duration
	onPriceUnits map[string][]*Historian
	onOrderUnits map[string][]*Historian
}

func NewHistorianStrategy(interval time.Duration) HistorianStrategy {
	return HistorianStrategy{
		Interval:     interval,
		onPriceUnits: make(map[string][]*Historian),
		onOrderUnits: make(map[string][]*Historian),
	}
}

func (r *HistorianStrategy) BindOnPrice(unit *Historian) {
}

// +----------+
// | Strategy |
// +----------+

func (r *HistorianStrategy) Init(k *Keep, e exchange.IBotExchange) error {
	r.onPriceUnits[e.GetName()] = make([]*Historian, 0)
	r.onOrderUnits[e.GetName()] = make([]*Historian, 0)

	return nil
}

func (r *HistorianStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return nil
}

func (r *HistorianStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	return fire(r.onPriceUnits, e, x)
}

func (r *HistorianStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return nil
}

func (r *HistorianStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return nil
}

func (r *HistorianStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return fire(r.onOrderUnits, e, x)
}

func (r *HistorianStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return nil
}

func (r *HistorianStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return nil
}

func (r *HistorianStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (r *HistorianStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	return nil
}

func fire(units map[string][]*Historian, e exchange.IBotExchange, x interface{}) error {
	name := e.GetName()
	for _, unit := range units[name] {
		unit.Push(x)
		unit.F(unit.state)
	}

	return nil
}
