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

var (
	ErrWebsocketNotSupported = errors.New("websocket not supported")
	ErrWebsocketNotEnabled   = errors.New("websocket is not enabled")
)

func logError(method string, data interface{}, err error) {
}

// openWebsocket resembles
// github.com/thrasher-copr/gocryptotrader.engine.websocketRoutineManager.websocketRoutine.
func openWebsocket(e exchange.IBotExchange) (*stream.Websocket, error) {
	// Check whether websocket is enabled.
	if !e.IsWebsocketEnabled() {
		return nil, ErrWebsocketNotEnabled
	}

	// Check whether websocket is supported.
	if !e.SupportsWebsocket() {
		return nil, ErrWebsocketNotSupported
	}

	// Instantiate a websocket.
	ws, err := e.GetWebsocket()
	if err != nil {
		return nil, err
	}

	// Connect.
	if !ws.IsConnecting() && !ws.IsConnected() {
		err = ws.Connect()
		if err != nil {
			return nil, err
		}

		err = ws.FlushChannels()
		if err != nil {
			return nil, err
		}
	}

	return ws, nil
}

// Stream resembles
// github.com/thrasher-copr/gocryptotrader.engine.websocketRoutineManager.WebsocketDataHandler.
func Stream(k *Keep, e exchange.IBotExchange, s Strategy) error {
	ws, err := openWebsocket(e)
	if err != nil {
		return err
	}

	// Init strategy.
	s.Init(k, e)

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
			logError("OnFunding", data, s.OnFunding(k, e, x))
		case *ticker.Price:
			logError("OnPrice", data, s.OnPrice(k, e, *x))
		case stream.KlineData:
			logError("OnKline", data, s.OnKline(k, e, x))
		case *orderbook.Base:
			logError("OnOrderBook", data, s.OnOrderBook(k, e, *x))
		case *order.Detail:
			logError("OnOrderPlace", data, s.OnOrder(k, e, *x))
		case *order.Modify:
			logError("OnModify", data, s.OnModify(k, e, *x))
		case order.ClassificationError:
			log.Warn().
				Str("exchange", x.Exchange).
				Str("OrderID", x.OrderID).
				Err(x.Err).
				Msg(Location())

			if x.Err == nil {
				panic("unexpected an error")
			}

			return x.Err
		case stream.UnhandledMessageWarning:
			log.Warn().
				Str("message", x.Message).
				Str("what", "unknown message").
				Msg(Location())
		case account.Change:
			logError("OnBalanceChange", data, s.OnBalanceChange(k, e, x))
		default:
			log.Debug().
				// Fields(map[string]interface{}{"data": data}).
				Str("type", fmt.Sprintf("%T", x)).
				Str("what", "unknown type").
				Msg(Location())
		}
	}

	// Deinit strategy.
	s.Deinit(k, e)

	panic("unexpected end of channel")
}
