package dola

import (
	"errors"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var ErrCreatorNotRegistered = errors.New("exchange creator not registered")

type (
	ExchangeCreatorFunc func() (exchange.IBotExchange, error)
	ExchangeFactory     map[string]ExchangeCreatorFunc
)

func (e ExchangeFactory) Register(name string, fn ExchangeCreatorFunc) {
	e[name] = fn
}

// NewExchangeByName implements gocryptotrader/engine.CustomExchangeBuilder.
func (e ExchangeFactory) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	fn, ok := e[name]
	if !ok {
		return nil, ErrCreatorNotRegistered
	}

	return fn()
}
