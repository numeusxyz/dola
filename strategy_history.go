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
	epoch int64
	state Array
}

func NewHistorian(interval time.Duration, stateLength int, f func(Array)) Historian {
	state := NewCircularArray(stateLength)

	return Historian{
		f:        f,
		interval: interval,
		epoch:    0,
		state:    &state,
	}
}

func (u *Historian) Push(x interface{}) {
	u.state.(*CircularArray).Push(x)
}

func (u *Historian) Update(now time.Time, x interface{}) {
	// If there is an interval specified, we should update once each interval.
	if u.interval != 0 {
		// Compute the current epoch.
		epoch := now.UnixNano() / u.interval.Nanoseconds()

		// If we're in the same epoch as the last update, return.
		if u.epoch == epoch {
			return
		}

		// // If this is the first ever update, just assign the epoch and move on.
		// // Otherwise the first update would be imbalanced.
		// if u.epoch == 0 {
		// 	u.epoch = epoch

		// 	return
		// }

		// We move on with the state update.
		u.epoch = epoch
	}

	u.state.(*CircularArray).Push(x)
	u.f(u.state)
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
	// some exchanges (eg. Kraken) don't provide the update timestamp so we fallback to `now` when
	// unavailable
	lastUpdated := x.LastUpdated
	if lastUpdated.IsZero() {
		lastUpdated = time.Now()
	}

	return fire(r.onPriceUnits, e, lastUpdated, x)
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

func (r *HistoryStrategy) Deinit(k *Keep, e exchange.IBotExchange) error {
	return nil
}

func fire(units map[string][]*Historian, e exchange.IBotExchange, now time.Time, x interface{}) error {
	name := e.GetName()

	// MT note: if historians do not get added and removed dynamically, this method is
	// completely safe to be used in a MT environment, because:
	//   1. reading (without concurrent writing) a map is MT-safe,
	//   2. all On*() events for a single exchange are invoked from the same thread.
	for _, unit := range units[name] {
		unit.Update(now, x)
	}

	return nil
}
