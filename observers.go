package dola

import (
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type OnFilledObserver interface {
	OnFilled(k *Keep, e exchange.IBotExchange, x order.Detail)
}

// +-------+
// | Slots |
// +-------+

type Slots struct {
	OnFilledSlot func(k *Keep, e exchange.IBotExchange, x order.Detail)
}

// OnFilled implements OnFilledObserver.
func (s Slots) OnFilled(k *Keep, e exchange.IBotExchange, x order.Detail) {
	if s.OnFilledSlot != nil {
		s.OnFilledSlot(k, e, x)
	}
}
