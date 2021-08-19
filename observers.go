package dola

import (
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type OnFilledObserver interface {
	OnFilled(e exchange.IBotExchange, x order.Detail)
}
