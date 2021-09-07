package dola

import (
	"errors"
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
	Last() interface{}

	Floats() []float64
	LastFloat() float64
}

type Historian struct {
	// Stateless.
	f        func(state Array)
	interval time.Duration

	// Stateful.
	lastUpdate time.Time
	state      Array
}

func NewHistorian(interval time.Duration, stateLength int, f func(Array)) Historian {
	state := NewCircularArray(stateLength)

	return Historian{
		f:          f,
		interval:   interval,
		lastUpdate: time.Time{},
		state:      &state,
	}
}

func (u *Historian) Push(x interface{}) {
	u.state.(*CircularArray).Push(x)
}

func (u *Historian) Update(now time.Time, x interface{}) {
	if u.lastUpdate.Add(u.interval).Before(now) {
		u.lastUpdate = now

		u.state.(*CircularArray).Push(x)
		u.f(u.state)
	}
}

// Floats returns the State array, but casted to []float64.
func (u *Historian) Floats() []float64 {
	return u.state.Floats()
}

// +-----------------+
// | HistoryStrategy |
// +-----------------+

var ErrUnknownEvent = errors.New("unknown event")

type HistoryStrategy struct {
	onPriceUnits map[string][]*Historian
	onOrderUnits map[string][]*Historian
}

func NewHistoryStrategy() HistoryStrategy {
	return HistoryStrategy{
		onPriceUnits: make(map[string][]*Historian),
		onOrderUnits: make(map[string][]*Historian),
	}
}

func (r *HistoryStrategy) BindOnPrice(unit *Historian) {
}

func (r *HistoryStrategy) AddHistorian(
	exchangeName,
	eventName string,
	interval time.Duration,
	stateLength int,
	f func(Array),
) error {
	historian := NewHistorian(interval, stateLength, f)

	switch eventName {
	case "OnPrice":
		xs := r.onPriceUnits[exchangeName]
		r.onPriceUnits[exchangeName] = append(xs, &historian)
	case "OnOrder":
		xs := r.onOrderUnits[exchangeName]
		r.onOrderUnits[exchangeName] = append(xs, &historian)
	default:
		return ErrUnknownEvent
	}

	return nil
}

// +----------+
// | Strategy |
// +----------+

func (r *HistoryStrategy) Init(k *Keep, e exchange.IBotExchange) error {
	r.onPriceUnits[e.GetName()] = make([]*Historian, 0)
	r.onOrderUnits[e.GetName()] = make([]*Historian, 0)

	return nil
}

func (r *HistoryStrategy) OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error {
	return nil
}

func (r *HistoryStrategy) OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error {
	return fire(r.onPriceUnits, e, x.LastUpdated, x)
}

func (r *HistoryStrategy) OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error {
	return nil
}

func (r *HistoryStrategy) OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error {
	return nil
}

func (r *HistoryStrategy) OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error {
	return fire(r.onOrderUnits, e, x.Date, x)
}

func (r *HistoryStrategy) OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error {
	return nil
}

func (r *HistoryStrategy) OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error {
	return nil
}

func (r *HistoryStrategy) OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error {
	return nil
}

func (r *HistoryStrategy) OnEvent(k *Keep, x interface{}) error {
	return nil
}

func (r *HistoryStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	return nil
}

func fire(units map[string][]*Historian, e exchange.IBotExchange, now time.Time, x interface{}) error {
	name := e.GetName()

	// MT note: if historians do not get added and removed dynamically, this methodis
	// completely fine, because:
	//   1. reading from a map is MT-safe,
	//   2. all On*() events for a singleexchange are invoked from the same thread.
	for _, unit := range units[name] {
		unit.Update(now, x)
	}

	return nil
}
