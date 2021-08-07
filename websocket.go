package dola

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func Stream(k *Keep, e exchange.IBotExchange, s Strategy) error {
	// Check whether websocket is enabled.
	if !e.SupportsWebsocket() || !e.IsWebsocketEnabled() {
		return errors.New("exchange either does not support websocket or is websocket is not enabled")
	}

	// Instantiate a websocket.
	ws, err := e.GetWebsocket()
	if err != nil {
		return err
	}

	// Connect.
	if !ws.IsConnecting() && !ws.IsConnected() {
		err = ws.Connect()
		if err != nil {
			return err
		}

		err = ws.FlushChannels()
		if err != nil {
			return err
		}
	}

	// This goroutine never, I repeat, *never* finishes.
	for data := range ws.ToRoutine {
		switch x := data.(type) {
		case string:
			log.Warn().
				Str("type", x).
				Str("what", "unknown string").
				Msg(Location())
		case error:
			return x
		case stream.FundingData:
			s.OnFunding(k, e, x)
		case *ticker.Price:
			s.OnPrice(k, e, *x)
		case stream.KlineData:
			s.OnKline(k, e, x)
		case *orderbook.Base:
			s.OnOrderBook(k, e, *x)
		case *order.Detail:
			s.OnOrder(k, e, *x)
		case *order.Modify:
			s.OnModify(k, e, *x)
		case order.ClassificationError:
			log.Warn().
				Str("exchange", x.Exchange).
				Str("OrderID", x.OrderID).
				Err(x.Err).
				Msg(Location())
			if x.Err == nil {
				panic("expected an error")
			}
			return x.Err
		case stream.UnhandledMessageWarning:
			log.Warn().
				Str("message", x.Message).
				Str("what", "unknown message").
				Msg(Location())
		case account.Change:
			s.OnBalanceChange(k, e, x)
		// case binance.wsAccountPosition:
		default:
			log.Warn().
				// Fields(map[string]interface{}{"data": data}).
				Str("type", fmt.Sprintf("%T", x)).
				Str("what", "unknown type").
				Msg(Location())
		}
	}

	// Unreachable since ws.ToRoutine NEVER gets closed.
	panic("unexpected")
}
