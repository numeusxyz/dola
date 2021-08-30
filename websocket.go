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

func Stream(k *Keep, e exchange.IBotExchange, s Strategy) error {
	ws, err := openWebsocket(e)
	if err != nil {
		return err
	}

	// Init strategy.
	if err := s.Init(k, e); err != nil {
		return err
	}

	// This goroutine never, I repeat, *never* finishes.
	for data := range ws.ToRoutine {
		if err := handleData(k, e, s, data); err != nil {
			return err
		}
	}

	// Deinit strategy.
	if err := s.Deinit(k, e); err != nil {
		return err
	}

	panic("unexpected end of channel")
}

// handleData resembles github.com/thrasher-corp/gocryptotrader.engine.websocketRoutineManager.WebsocketDataHandler.
//
//nolint:cyclop
func handleData(k *Keep, e exchange.IBotExchange, s Strategy, data interface{}) error {
	switch x := data.(type) {
	case string:
		unhandledType(data, true)
	case error:
		return x
	case stream.FundingData:
		handleError("OnFunding", s.OnFunding(k, e, x))
	case *ticker.Price:
		handleError("OnPrice", s.OnPrice(k, e, *x))
	case stream.KlineData:
		handleError("OnKline", s.OnKline(k, e, x))
	case *orderbook.Base:
		handleError("OnOrderBook", s.OnOrderBook(k, e, *x))
	case *order.Detail:
		k.OnOrder(e, *x)
		handleError("OnOrder", s.OnOrder(k, e, *x))
	case *order.Modify:
		handleError("OnModify", s.OnModify(k, e, *x))
	case order.ClassificationError:
		unhandledType(data, true)

		if x.Err == nil {
			panic("unexpected error")
		}

		return x.Err
	case stream.UnhandledMessageWarning:
		unhandledType(data, true)
	case account.Change:
		handleError("OnBalanceChange", s.OnBalanceChange(k, e, x))
	default:
		handleError("OnUnrecognized", s.OnUnrecognized(k, e, data))
	}

	return nil
}

func handleError(method string, err error) {
	if err != nil {
		What(log.Warn().
			Err(err).
			Str("method", method),
			"method failed")
	}
}

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

func unhandledType(data interface{}, warn bool) {
	e := log.Debug()

	if warn {
		e = log.Warn()
	}

	t := fmt.Sprintf("%T", data)
	e = e.Interface("data", data).Str("type", t)

	What(e, "unhandled type")
}
