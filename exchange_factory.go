package dola

import (
	"errors"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var ErrCreatorNotRegistered = errors.New("exchange creator not registered")

type ExchangeCreatorFuncK func(k *Keep) (exchange.IBotExchange, error)

// ExchangeCreatorFunc type is the func type being stored in the factory map
// GCT doesn't allow us to carry context.
type ExchangeCreatorFunc func() (exchange.IBotExchange, error)

type ExchangeFactory map[string]ExchangeCreatorFunc

func (e ExchangeFactory) Register(name string, fn ExchangeCreatorFunc) {
	e[name] = fn
}

// NewExchangeByName is satisfying GCT's interface CustomExchangeBuilder.
func (e ExchangeFactory) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	fn, ok := e[name]
	if !ok {
		return nil, ErrCreatorNotRegistered
	}

	return fn()
}
