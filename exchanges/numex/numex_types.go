package numex

import (
	"time"
)

const wsRateLimitMilliseconds = 250

// OHLCVData stores historical OHLCV data
type OHLCVData struct {
	Close     float64   `json:"close"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Open      float64   `json:"open"`
	StartTime time.Time `json:"startTime"`
	Time      int64     `json:"time"`
	Volume    float64   `json:"volume"`
}

// WsRequest defines the payload of client requests through the websocket connection
type WsRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	ID     int64    `json:"id"`
}

// WsEvent defines the payload of server events through the websocket connection
type WsEvent struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
}

type WsOrder struct {
	ID        string  `json:"id"`            // The id of the order
	Operation int     `json:"operation"`     // Type of order operation (eg. market, cancel, modify, status, etc)
	Direction int     `json:"direction"`     // Whether this order is buying (bid) or selling (ask)
	Asset     int     `json:"asset"`         // Type of Asset (Spot, etc)
	Pair      string  `json:"pair"`          // Currency pair (eg. BTC-USDT)
	Amount    float64 `json:"amount,string"` // Amount to buy
	Price     float64 `json:"price,string"`  // Price
	Timestamp int     `json:"timestamp"`     // Timestamp in nanoseconds
}

type WsFill struct {
	ID           string    `json:"id"`
	Asset        int       `json:"asset"`
	Pair         string    `json:"pair"`
	Amount       float64   `json:"amount,string"`
	Price        float64   `json:"price,string"`
	Timestamp    int       `json:"timestamp"`
	Participants []WsOrder `json:"participants"`
	Closed       []WsOrder `json:"closed"`
}

type WsTicker struct {
	Symbol         string  `json:"symbol"`
	PrevClosePrice float64 `json:"prevClosePrice,string"`
	LastPrice      float64 `json:"lastPrice,string"`
	LastQty        float64 `json:"lastQty,string"`
	BidPrice       float64 `json:"bestBidPrice,string"`
	AskPrice       float64 `json:"bestAskPrice,string"`
	OpenPrice      float64 `json:"openPrice,string"`
	HighPrice      float64 `json:"highPrice,string"`
	LowPrice       float64 `json:"lowPrice,string"`
	Volume         float64 `json:"volume,string"`
	QuoteVolume    float64 `json:"quoteVolume,string"`
	LastUpdate     int64   `json:"lastUpdate,string"`
}

// OrderbookItem stores an individual orderbook item
type OrderbookItem struct {
	Price    float64
	Quantity float64
}

// OrderBook actual structured data that can be used for orderbook
type OrderBook struct {
	Symbol       string
	LastUpdateID int64
	Code         int
	Msg          string
	Bids         []OrderbookItem
	Asks         []OrderbookItem
}

type Symbol struct {
	Symbol               string `json:"symbol"`
	Status               string `json:"status"`
	BaseAsset            string `json:"baseAsset"`
	BaseAssetPrecision   int    `json:"baseAssetPrecision"`
	QuoteAsset           string `json:"quoteAsset"`
	QuotePrecision       int    `json:"quotePrecision"`
	IsSpotTradingAllowed bool   `json:"isSpotTradingAllowed"`
}

type ExchangeInfo struct {
	Symbols []Symbol `json:"symbols"`
}

// Ticker contains statistics for the last 24 hours trade
type Ticker struct {
	Symbol         string  `json:"symbol"`
	PrevClosePrice float64 `json:"prevClosePrice,string"`
	LastPrice      float64 `json:"lastPrice,string"`
	LastQty        float64 `json:"lastQty,string"`
	BidPrice       float64 `json:"bidPrice,string"`
	AskPrice       float64 `json:"askPrice,string"`
	OpenPrice      float64 `json:"openPrice,string"`
	HighPrice      float64 `json:"highPrice,string"`
	LowPrice       float64 `json:"lowPrice,string"`
	Volume         float64 `json:"volume,string"`
	QuoteVolume    float64 `json:"quoteVolume,string"`
}

type Order struct {
	Direction string  `json:"direction"`     // Whether this order is buying (bid), selling (ask) or others
	Amount    float64 `json:"amount,string"` // Amount to trade
	Price     float64 `json:"price"`         // Price in base currency
}

type OrderbookData struct {
	Depth int     `json:"depth"` // The asset (eg. Spot)
	Book  []Order `json:"book"`  // The currency pair (eg. BTC-USDT, ETH-USDT)
}
