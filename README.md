# Dola

> Що се рекло от три наречници,  
> що се рекло и се извършило...

Dola is a cryptocurrency trading library that provides:

* exchanges integration (via [github.com/thrasher-corp/gocryptotrader](https://github.com/thrasher-corp/gocryptotrader)),
* event-driven strategies,
* utilities and more.

## Exchanges

Dola expects to find a configuration file at `~/.dola/config.json`.
See an example
[here](https://github.com/thrasher-corp/gocryptotrader/blob/master/config_example.json).

**NB: Dola needs just the `exchanges` section.**

## Strategies

Each strategy is an implementation of the following interface. Dola
has the responsibility of invoking all of the `Strategy` methods.
`Strategy.On*` methods are invoked whenever there is new data from one
of the registered exchanges. If you need a strategy that supports just
one exchange (out of many), take a look at `DedicatedStrategy`.

```go
type Strategy interface {
	Init(k *Keep, e exchange.IBotExchange) error
	OnFunding(k *Keep, e exchange.IBotExchange, x stream.FundingData) error
	OnPrice(k *Keep, e exchange.IBotExchange, x ticker.Price) error
	OnKline(k *Keep, e exchange.IBotExchange, x stream.KlineData) error
	OnOrderBook(k *Keep, e exchange.IBotExchange, x orderbook.Base) error
	OnOrder(k *Keep, e exchange.IBotExchange, x order.Detail) error
	OnModify(k *Keep, e exchange.IBotExchange, x order.Modify) error
	OnBalanceChange(k *Keep, e exchange.IBotExchange, x account.Change) error
	OnUnrecognized(k *Keep, e exchange.IBotExchange, x interface{}) error
	Deinit(k *Keep, e exchange.IBotExchange) error
}
```

## Example

```go
package main

import (
	"github.com/numus-digital/dola"
)

func main() {
	keep, _ := dola.NewKeepBuilder().Build()
	keep.Root.Add("verbose", dola.VerboseStrategy{})
	keep.Run()
}

```

### Augment config

```go
keep, _ := dola.NewKeepBuilder().Augment(func (c *config.Config) erro {
    doSomething(c)
}).Build()
```

### Build custom exchanges

```go
creator := func() (exchange.IBotExchange, error) {
  return NewCustomExchange()
}
keep, _ := dola.NewKeepBuilder().CustomExchange(name, creator).Build()
```
