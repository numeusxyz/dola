package numex

import (
	"errors"
	"net/url"
	"strconv"
	"time"
	"context"
    "net/http"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Numex is the overarching type across this package
type Numex struct {
	exchange.Base
}

const (
    numexAPIURL     = "http://localhost:8080"
	numexAPIVersion = "v1"

    numexWSAPIURL     = "http://localhost:8080/stream"
	// Public endpoints

	// Authenticated endpoints
)

var (
	errStartTimeCannotBeAfterEndTime = errors.New("start timestamp cannot be after end timestamp")

	validResolutionData = []int64{15, 60, 300, 900, 3600, 14400, 86400}
)

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (n *Numex) GetExchangeInfo() (ExchangeInfo, error) {
	var resp ExchangeInfo
	return resp, n.SendHTTPRequest(exchange.RestSpotSupplementary, "/info", &resp)
}

// GetTicker returns the ticker data for the last 24 hrs
func (n *Numex) GetTicker(p currency.Pair, a asset.Item) (Ticker, error) {
	var resp Ticker
	params := url.Values{}
    params.Set("asset", a.String())
    params.Set("pair", p.String())
	path := "/ticker" + "?" + params.Encode()
	return resp, n.SendHTTPRequest(exchange.RestSpotSupplementary, path, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (n *Numex) GetOrderBook(p currency.Pair, a asset.Item) (OrderBook, error) {
	var orderbook OrderBook

	params := url.Values{}
    params.Set("asset", a.String())
    params.Set("pair", p.String())
	path := "/orderbook" + "?" + params.Encode()

	var resp OrderbookData
	if err := n.SendHTTPRequest(exchange.RestSpotSupplementary, path, &resp); err != nil {
		return orderbook, err
	}

	for _, order := range resp.Book {
        item := OrderbookItem{
			Price:    order.Price,
			Quantity: float64(order.Amount),
		}
        if order.Direction == "bid" {
            orderbook.Bids = append(orderbook.Bids, item)
        } else if order.Direction == "ask" {
            orderbook.Asks = append(orderbook.Asks, item)
        }
	}

    // TODO: add a last update id
	orderbook.LastUpdateID = 0
	return orderbook, nil
}

// GetHistoricalData gets historical OHLCV data for a given market pair
func (n *Numex) GetHistoricalData(marketName string, timeInterval, limit int64, startTime, endTime time.Time) ([]OHLCVData, error) {
	if marketName == "" {
		return nil, errors.New("a market pair must be specified")
	}

	err := checkResolution(timeInterval)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("resolution", strconv.FormatInt(timeInterval, 10))
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errStartTimeCannotBeAfterEndTime
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	// resp := struct {
	// 	Data []OHLCVData `json:"result"`
	// }{}
	// endpoint := common.EncodeURLValues(fmt.Sprintf(getHistoricalData, marketName), params)
	// return resp.Data, f.SendHTTPRequest(exchange.RestSpot, endpoint, &resp)

	var dummy []OHLCVData
	dummy = append(dummy, OHLCVData{
		Close:     1000,
		High:      2000,
		Low:       500,
		Open:      1000,
		StartTime: startTime,
		Time:      startTime.UnixNano(),
		Volume:    1,
	})
	return dummy, nil
}

// Helper functions
func checkResolution(res int64) error {
	for x := range validResolutionData {
		if validResolutionData[x] == res {
			return nil
		}
	}
	return errors.New("resolution data is a mandatory field and the data provided is invalid")
}

// SendHTTPRequest sends an unauthenticated request
func (n *Numex) SendHTTPRequest(ePath exchange.URL, path string, result interface{}) error {
	endpointPath, err := n.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       n.Verbose,
		HTTPDebugging: n.HTTPDebugging,
		HTTPRecording: n.HTTPRecording}

	return n.SendPayload(context.Background(), request.Unset, func() (*request.Item, error) {
		return item, nil
	})
}

