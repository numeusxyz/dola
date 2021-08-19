package dola

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type OrderKey struct {
	ExchangeName string
	OrderID      string
}

type OrderValue struct {
	SubmitResponse order.SubmitResponse
	UserData       interface{}
}

type OrderRegistry struct {
	m      sync.Map
	length int32
}

// Store saves order details.  If such an order exists
// (matched by exchange name and order ID), false is returned.
func (r *OrderRegistry) Store(exchangeName string, response order.SubmitResponse, userData interface{}) bool {
	key := OrderKey{
		ExchangeName: exchangeName,
		OrderID:      response.OrderID,
	}
	value := OrderValue{
		SubmitResponse: response,
		UserData:       userData,
	}
	_, loaded := r.m.LoadOrStore(key, value)

	if !loaded {
		// If not loaded, then it's stored, so length++.
		atomic.AddInt32(&r.length, 1)
	}

	return !loaded
}

func (r *OrderRegistry) GetOrderValue(exchangeName, orderID string) (OrderValue, bool) {
	key := OrderKey{
		ExchangeName: exchangeName,
		OrderID:      orderID,
	}

	var (
		loaded  bool
		ok      bool
		pointer interface{}
		value   OrderValue
	)

	if pointer, loaded = r.m.Load(key); loaded {
		value, ok = pointer.(OrderValue)
		if !ok {
			log.Fatalf("have %T, want OrderValue", pointer)
		}
	}

	return value, loaded
}

func (r *OrderRegistry) Length() int {
	return int(atomic.LoadInt32(&r.length))
}
