package dola

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/numus-digital/dola/exchanges/numex"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

type ExchangeBuilder struct {
}

// vars related to exchange functions
var (
	ErrExchangeNotFound = errors.New("exchange not found")
)

func (n ExchangeBuilder) NewExchangeByName(name string) (exchange.IBotExchange, error) {
	var exch exchange.IBotExchange

	found, err := regexp.MatchString("numex.", name)
	if !found || err != nil {
		return nil, fmt.Errorf("%s, %w", name, ErrExchangeNotFound)
	}

	exch = new(numex.Numex)
	b := exch.GetBase()
	b.Name = name

	return exch, nil
}
