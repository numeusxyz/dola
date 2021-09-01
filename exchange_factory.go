package dola

import (
	"errors"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

var ErrCreatorNotRegistered = errors.New("exchange creator not registered")

type CreatorFunc func(k *Keep, name string) (exchange.IBotExchange, error)

type ExchangeFactory map[string]CreatorFunc

func (e ExchangeFactory) Register(name string, fn CreatorFunc) {
	e[name] = fn
}

func (e ExchangeFactory) Create(k *Keep, name string) (exchange.IBotExchange, error) {
	fn, ok := e[name]

	if !ok {
		return nil, ErrCreatorNotRegistered
	}

	return fn(k, name)
}
