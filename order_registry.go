package dola

import (
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
	Length int32
}

func NewOrderRegistry() OrderRegistry {
	return OrderRegistry{sync.Map{}, 0}
}

// OnSubmit saves order details.  If such an order exists (matched by exchange name and
// order ID), false is returned.
func (r *OrderRegistry) OnSubmit(exchangeName string, response order.SubmitResponse, userData interface{}) bool {
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
		atomic.AddInt32(&r.Length, 1)
	}

	return !loaded
}

func (r *OrderRegistry) GetOrderValue(exchangeName, orderID string) (OrderValue, bool) {
	key := OrderKey{
		ExchangeName: exchangeName,
		OrderID:      orderID,
	}

	if p, ok := r.m.Load(key); ok {
		return p.(OrderValue), ok
	}

	return OrderValue{}, false // nolint: exhaustivestruct
}

func (r *OrderRegistry) MapUnsafe() *sync.Map {
	return &r.m
}
