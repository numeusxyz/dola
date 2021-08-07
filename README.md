# Dola

> Що се рекло от три наречници,  
> що се рекло и се извършило...

Dola is a cryptocurrency trading library that provides:

* exchange integrations (via [github.com/thrasher-corp/gocryptotrader](https://github.com/thrasher-corp/gocryptotrader)),
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
	Init(e exchange.IBotExchange) error
	OnFunding(e exchange.IBotExchange, x stream.FundingData) error
	OnPrice(e exchange.IBotExchange, x ticker.Price) error
	OnKline(e exchange.IBotExchange, x stream.KlineData) error
	OnOrderBook(e exchange.IBotExchange, x orderbook.Base) error
	OnOrder(e exchange.IBotExchange, x order.Detail) error
	OnModify(e exchange.IBotExchange, x order.Modify) error
	OnBalanceChange(e exchange.IBotExchange, x account.Change) error
	Deinit(e exchange.IBotExchange) error
}
```

## Example

```go
package main

import (
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/ydm/dola"
)

func main() {
	keep, err := dola.NewKeep(engine.Settings{})
	if err != nil {
		panic(err)
	}
	keep.Root.Add("verbose", dola.VerboseStrategy{})
	keep.Run()
}

```
