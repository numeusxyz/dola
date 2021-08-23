package numex

import (
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (n *Numex) GetDefaultConfig() (*config.ExchangeConfig, error) {
	n.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = n.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = n.BaseCurrencies

	n.SetupDefaults(exchCfg)

	if n.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := n.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Numex
func (n *Numex) SetDefaults() {
	n.Enabled = true
	n.Verbose = true
	// n.API.CredentialsValidator.RequiresKey = true
	// n.API.CredentialsValidator.RequiresSecret = true

	// If using only one pair format for request and configuration, across all
	// supported asset types either SPOT and FUTURES etc. You can use the
	// example below:

	// Request format denotes what the pair as a string will be, when you send
	// a request to an exchange.
	requestFmt := &currency.PairFormat{
		/*Set pair request formatting details here for e.g.*/
		Uppercase: true,
		Delimiter: ":",
	}
	// Config format denotes what the pair as a string will be, when saved to
	// the config.json file.
	configFmt := &currency.PairFormat{
		Delimiter: currency.DashDelimiter,
		Uppercase: true,
	}

	err := n.SetGlobalPairsManager(requestFmt, configFmt /*multiple assets can be set here using the asset package ie asset.Spot*/)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// If assets require multiple differences in formating for request and
	// configuration, another exchange method can be be used e.g. futures
	// contracts require a dash as a delimiter rather than an underscore. You
	// can use this example below:

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
	}

	err = n.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	n.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				TickerFetching:         true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrder:               true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.FifteenSecond.Word(): true,
					kline.OneMin.Word():        true,
					kline.FiveMin.Word():       true,
					kline.FifteenMin.Word():    true,
					kline.OneHour.Word():       true,
					kline.FourHour.Word():      true,
					kline.OneDay.Word():        true,
				},
				ResultLimit: 5000,
			},
		},
	}
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	n.Requester = request.New(n.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	// NOTE: SET THE URLs HERE
	n.API.Endpoints = n.NewEndpoints()
	n.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      numexAPIURL,
		exchange.WebsocketSpot: numexWSAPIURL,
	})
	n.Websocket = stream.New()
	n.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	n.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	n.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (n *Numex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		n.SetEnabled(false)
		return nil
	}

	fmt.Printf("Numex.setup\n")
	n.SetupDefaults(exch)

	wsRunningEndpoint, err := n.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	// If websocket is supported, please fill out the following
	err = n.Websocket.Setup(
		&stream.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       numexWSAPIURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       wsRunningEndpoint,
			Connector:                        n.WsConnect,
			Subscriber:                       n.Subscribe,
			UnSubscriber:                     n.Unsubscribe,
			GenerateSubscriptions:            n.GenerateSubscriptions,
			Features:                         &n.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}

	return n.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            wsRateLimitMilliseconds,
	})
}

// Start starts the Numex go routine
func (n *Numex) Start(wg *sync.WaitGroup) {
	fmt.Printf("Numex.Start\n")
	wg.Add(1)
	go func() {
		n.Run()
		wg.Done()
	}()
}

// Run implements the Numex wrapper
func (n *Numex) Run() {
	if n.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			n.Name,
			common.IsEnabled(n.Websocket.IsEnabled()))
		n.PrintEnabledPairs()
	}

	if !n.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := n.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			n.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (n *Numex) FetchTradablePairs(a asset.Item) ([]string, error) {
	fmt.Printf("Numex.FetchTradablePairs\n")
	if !n.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, n.Name)
	}

	format, err := n.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}

	var pairs []string
	switch a {
	case asset.Spot:
		info, err := n.GetExchangeInfo()
		if err != nil {
			return nil, err
		}
		for x := range info.Symbols {
			if info.Symbols[x].Status == "ACTIVE" {
				pair := info.Symbols[x].BaseAsset +
					format.Delimiter +
					info.Symbols[x].QuoteAsset
				if a == asset.Spot && info.Symbols[x].IsSpotTradingAllowed {
					pairs = append(pairs, pair)
				}
			}
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (n *Numex) UpdateTradablePairs(forceUpdate bool) error {
	fmt.Printf("Numex.UpdateTradablePairs\n")
	pairs, err := n.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return n.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (n *Numex) UpdateTicker(p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fmt.Printf("Numex.UpdateTicker\n")
	switch a {
	case asset.Spot:
		t, err := n.GetTicker(p, a)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         t.LastPrice,
			High:         t.HighPrice,
			Low:          t.LowPrice,
			Bid:          t.BidPrice,
			Ask:          t.AskPrice,
			Volume:       t.Volume,
			QuoteVolume:  t.QuoteVolume,
			Open:         t.OpenPrice,
			Close:        t.PrevClosePrice,
			Pair:         p,
			ExchangeName: n.Name,
			AssetType:    a,
		})
	default:
		return nil, fmt.Errorf("assetType not supported: %v", a)
	}
	return ticker.GetTicker(n.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (n *Numex) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fmt.Printf("Numex.FetchTicker\n")
	fPair, err := n.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(n.Name, fPair, assetType)
	if err != nil {
		return n.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (n *Numex) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fmt.Printf("Numex.FetchOrderbook\n")
	ob, err := orderbook.Get(n.Name, currency, assetType)
	if err != nil {
		return n.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (n *Numex) UpdateOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        n.Name,
		Pair:            p,
		Asset:           a,
		VerifyOrderbook: n.CanVerifyOrderbook,
	}

	ob, err := n.GetOrderBook(p, a)
	if err != nil {
		return book, err
	}

	for x := range ob.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: ob.Bids[x].Quantity,
			Price:  ob.Bids[x].Price,
		})
	}

	for x := range ob.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: ob.Asks[x].Quantity,
			Price:  ob.Asks[x].Price,
		})
	}

	err = book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(n.Name, p, a)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (n *Numex) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	fmt.Printf("Numex.UpdateAccountInfo\n")
	return account.Holdings{}, common.ErrNotYetImplemented
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (n *Numex) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	fmt.Printf("Numex.FetchAccountInfo\n")
	return account.Holdings{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (n *Numex) GetFundingHistory() ([]exchange.FundHistory, error) {
	fmt.Printf("Numex.GetFundingHistory\n")
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (n *Numex) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	fmt.Printf("Numex.GetWithdrawalsHistory\n")
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (n *Numex) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	fmt.Printf("Numex.GetRecentTrades\n")
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (n *Numex) GetHistoricTrades(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	fmt.Printf("Numex.GetHistoricTrades\n")
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (n *Numex) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	fmt.Printf("Numex.SubmitOrder\n")
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (n *Numex) ModifyOrder(action *order.Modify) (order.Modify, error) {
	fmt.Printf("Numex.ModifyOrder\n")
	// if err := action.Validate(); err != nil {
	// 	return "", err
	// }
	return *action, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (n *Numex) CancelOrder(ord *order.Cancel) error {
	fmt.Printf("Numex.CancelOrder\n")
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (n *Numex) CancelBatchOrders(orders []order.Cancel) (order.CancelBatchResponse, error) {
	fmt.Printf("Numex.CancelBatchOrders\n")
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (n *Numex) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	fmt.Printf("Numex.CancelAllOrders\n")
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (n *Numex) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	fmt.Printf("Numex.GetOrderInfo\n")
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (n *Numex) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	fmt.Printf("Numex.GetDepositAddress\n")
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (n *Numex) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (n *Numex) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (n *Numex) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (n *Numex) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (n *Numex) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	fmt.Printf("Numex.GetOrderHistory\n")
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (n *Numex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	fmt.Printf("Numex.GetFeeByType\n")
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (n *Numex) ValidateCredentials(assetType asset.Item) error {
	_, err := n.UpdateAccountInfo(assetType)
	return n.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (n *Numex) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	fmt.Printf("Numex.GetHistoricCandles\n")
	return kline.Item{}, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (n *Numex) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	fmt.Printf("Numex.GetHistoricCandlesExtended\n")
	if err := n.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: n.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	dates, err := kline.CalculateCandleDateRanges(start, end, interval, n.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := n.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range dates.Ranges {
		var ohlcData []OHLCVData
		ohlcData, err = n.GetHistoricalData(formattedPair.String(),
			int64(interval.Duration().Seconds()),
			int64(n.Features.Enabled.Kline.ResultLimit),
			dates.Ranges[x].Start.Time, dates.Ranges[x].End.Time)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range ohlcData {
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   ohlcData[i].StartTime,
				Open:   ohlcData[i].Open,
				High:   ohlcData[i].High,
				Low:    ohlcData[i].Low,
				Close:  ohlcData[i].Close,
				Volume: ohlcData[i].Volume,
			})
		}
	}

	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", n.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
