package dola

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ftx"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func logError(method string, data interface{}, err error) {
}

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
			logError("OnFunding", data, s.OnFunding(k, e, x))
		case *ticker.Price:
			logError("OnPrice", data, s.OnPrice(k, e, *x))
		case stream.KlineData:
			logError("OnKline", data, s.OnKline(k, e, x))
		case *orderbook.Base:
			logError("OnOrderBook", data, s.OnOrderBook(k, e, *x))
		case *order.Detail:
			copy := *x
			if copy.Status == order.New {
				logError("OnOrderPlace", data, s.OnOrderPlace(k, e, copy))
			} else {
				logError("OnOrderPlace", data, s.OnOrder(k, e, *x))
			}
		case *order.Modify:
			logError("OnModify", data, s.OnModify(k, e, *x))
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
			logError("OnBalanceChange", data, s.OnBalanceChange(k, e, x))
		// case binance.wsAccountPosition:
		//
		// Order filling is now supported just for FTX.
		// Support for other exchanges should be added
		// manually here.
		case ftx.WsFills:
			err = s.OnTrade(k, e, Trade{
				Timestamp:     x.Time,
				BaseCurrency:  x.BaseCurrency,
				QuoteCurrency: x.QuoteCurrency,
				OrderID:       strconv.FormatInt(x.OrderID, 10),
				AveragePrice:  x.Price,
				Quantity:      x.Size,
				Fee:           x.Fee,
				FeeCurrency:   x.FeeCurrency,
			})
			logError("OnTrade", data, err)
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
