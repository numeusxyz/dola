package numex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	pingDelay = time.Minute * 9
)

// WsConnect initiates a websocket connection
func (n *Numex) WsConnect() error {
	if !n.Websocket.IsEnabled() || !n.IsEnabled() {
		return errors.New(stream.WebsocketNotEnabled)
	}

	var dialer websocket.Dialer
	dialer.HandshakeTimeout = n.Config.HTTPTimeout
	dialer.Proxy = http.ProxyFromEnvironment
	var err error

	err = n.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s",
			n.Name,
			err)
	}

	n.Websocket.Conn.SetupPingHandler(stream.PingHandler{
		UseGorillaHandler: true,
		MessageType:       websocket.PongMessage,
		Delay:             pingDelay,
	})

	go n.wsReadData()
	return nil
}

// GenerateSubscriptions generates the default subscription set
func (n *Numex) GenerateSubscriptions() ([]stream.ChannelSubscription, error) {
	var channels = []string{"@ticker", "@order"}
	var subscriptions []stream.ChannelSubscription
	assets := n.GetAssetTypes(true)
	for _, a := range assets {
		switch a {
		case asset.Spot:
			pairs, err := n.GetEnabledPairs(a)
			if err != nil {
				return nil, err
			}

			for _, p := range pairs {
				for _, c := range channels {
					subscriptions = append(subscriptions, stream.ChannelSubscription{
						Channel:  p.String() + c,
						Currency: p,
						Asset:    a,
					})
				}
			}
		}
	}
	return subscriptions, nil
}

// Subscribe subscribes to a set of channels
func (n *Numex) Subscribe(channelsToSubscribe []stream.ChannelSubscription) error {
	payload := WsRequest{
		Method: "SUBSCRIBE",
	}
	for i := range channelsToSubscribe {
		payload.Params = append(payload.Params, channelsToSubscribe[i].Channel)
		if i%50 == 0 && i != 0 {
			err := n.Websocket.Conn.SendJSONMessage(payload)
			if err != nil {
				return err
			}
			payload.Params = []string{}
		}
	}
	if len(payload.Params) > 0 {
		err := n.Websocket.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}
	}
	n.Websocket.AddSuccessfulSubscriptions(channelsToSubscribe...)
	return nil
}

// Unsubscribe unsubscribes from a set of channels
func (n *Numex) Unsubscribe(channelsToUnsubscribe []stream.ChannelSubscription) error {
	payload := WsRequest{
		Method: "UNSUBSCRIBE",
	}
	for i := range channelsToUnsubscribe {
		payload.Params = append(payload.Params, channelsToUnsubscribe[i].Channel)
		if i%50 == 0 && i != 0 {
			err := n.Websocket.Conn.SendJSONMessage(payload)
			if err != nil {
				return err
			}
			payload.Params = []string{}
		}
	}
	if len(payload.Params) > 0 {
		err := n.Websocket.Conn.SendJSONMessage(payload)
		if err != nil {
			return err
		}
	}
	n.Websocket.RemoveSuccessfulUnsubscriptions(channelsToUnsubscribe...)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (n *Numex) wsReadData() {
	n.Websocket.Wg.Add(1)
	defer n.Websocket.Wg.Done()

	for {
		resp := n.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		err := n.wsHandleData(resp.Raw)
		if err != nil {
			n.Websocket.DataHandler <- err
		}
	}
}
func (n *Numex) wsHandleData(data []byte) error {
	var e WsEvent
	err := json.Unmarshal(data, &e)
	if err != nil {
		return err
	}

	switch e.Type {
	case "ORDER":
		var o WsOrder
		err := json.Unmarshal(e.Data, &o)
		if err != nil {
			return err
		}
		// p, err := currency.NewPairFromString(o.Pair)
		// if err != nil {
		// 	return err
		// }
	case "FILL":
		var f WsFill
		err := json.Unmarshal(e.Data, &f)
		if err != nil {
			return err
		}
		p, err := currency.NewPairFromString(f.Pair)
		if err != nil {
			return err
		}
		return n.AddTradesToBuffer(trade.Data{
			CurrencyPair: p,
			Timestamp:    time.Unix(int64(f.Timestamp), 0),
			Price:        f.Price,
			Amount:       float64(f.Amount),
			Exchange:     n.Name,
			AssetType:    asset.Spot,
			TID:          f.ID,
		})
	case "TICKER":
		var t WsTicker
		err := json.Unmarshal(e.Data, &t)
		if err != nil {
			return err
		}
		p, err := currency.NewPairFromString(t.Symbol)
		if err != nil {
			return err
		}
		n.Websocket.DataHandler <- &ticker.Price{
			ExchangeName: n.Name,
			Open:         t.OpenPrice,
			Close:        t.PrevClosePrice,
			Volume:       t.Volume,
			QuoteVolume:  t.QuoteVolume,
			High:         t.HighPrice,
			Low:          t.LowPrice,
			Bid:          t.BidPrice,
			Ask:          t.AskPrice,
			Last:         t.LastPrice,
			LastUpdated:  time.Unix(0, int64(t.LastUpdate)),
			AssetType:    asset.Spot,
			Pair:         p,
		}
	}

	return nil
}
