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

// +--------------+
// | ReporterUnit |
// +--------------+

type Array interface {
	At(index int) interface{}
	Len() int
}

type ReporterUnit struct {
	Key   string
	F     func(state Array)
	state Array
}

func NewReporterUnit(key string, n int, f func(state Array)) ReporterUnit {
	state := NewCircularArray(n)

	return ReporterUnit{
		Key:   key,
		F:     f,
		state: &state,
	}
}

func (u *ReporterUnit) Push(x interface{}) {
	u.state.(*CircularArray).Push(x)
}

// Floats returns the State array, but casted to []float64.
func (u *ReporterUnit) Floats() []float64 {
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

func (u *ReporterUnit) Last() interface{} {
	index := u.state.Len() - 1

	return u.state.At(index)
}

// +-----------+
// | Historian |
// +-----------+

type Historian struct {
	Interval     time.Duration
	onPriceUnits map[string][]*ReporterUnit
	onOrderUnits map[string][]*ReporterUnit
}

func NewHistorian(interval time.Duration) Historian {
	return Historian{
		Interval:     interval,
		onPriceUnits: make(map[string][]*ReporterUnit),
		onOrderUnits: make(map[string][]*ReporterUnit),
	}
}

func (r *Historian) BindOnPrice(unit *ReporterUnit) {
}

// +------------------------------------+
// | Historian: Strategy implementation |
// +------------------------------------+

func (r *Historian) Init(k *Keep, e exchange.IBotExchange) error {
	r.onPriceUnits[e.GetName()] = make([]*ReporterUnit, 0)
	r.onOrderUnits[e.GetName()] = make([]*ReporterUnit, 0)

	return nil
}

func (r *Historian) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return nil
}

func (r *Historian) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	return fire(r.onPriceUnits, e, x)
}

func (r *Historian) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return nil
}

func (r *Historian) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return nil
}

func (r *Historian) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return fire(r.onOrderUnits, e, x)
}

func (r *Historian) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return nil
}

func (r *Historian) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return nil
}

func (r *Historian) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (r *Historian) Deinit(k *Keep, e exchange.IBotExchange) error {
	return nil
}

func fire(units map[string][]*ReporterUnit, e exchange.IBotExchange, x interface{}) error {
	name := e.GetName()
	for _, unit := range units[name] {
		unit.Push(x)
		unit.F(unit.state)
	}

	return nil
}
