package dola

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"go.uber.org/multierr"
)

// +-------------+
// | KeepBuilder |
// +-------------+

type (
	AugmentConfigFunc func(*config.Config) error
)

type KeepBuilder struct {
	augment  AugmentConfigFunc
	factory  ExchangeFactory
	settings engine.Settings
}

func NewKeepBuilder() *KeepBuilder {
	// nolint: exhaustivestruct
	return &KeepBuilder{
		augment: nil,
		factory: ExchangeFactory{},
	}
}

func (b *KeepBuilder) Augment(f AugmentConfigFunc) *KeepBuilder {
	b.augment = f

	return b
}

func (b *KeepBuilder) Settings(s engine.Settings) *KeepBuilder {
	b.settings = s

	return b
}

func (b *KeepBuilder) CustomExchange(name string, fn ExchangeCreatorFunc) *KeepBuilder {
	b.factory.Register(name, fn)

	return b
}

func (b *KeepBuilder) Build() (*Keep, error) {
	// Resolve path to config file.
	b.settings.ConfigFile = ConfigFile(b.settings.ConfigFile)

	filePath, err := config.GetAndMigrateDefaultPath(b.settings.ConfigFile)
	if err != nil {
		return nil, err
	}

	var (
		conf config.Config
		keep = &Keep{
			Config:          conf,
			ExchangeManager: *engine.SetupExchangeManager(),
			Root:            NewRootStrategy(),
			Settings:        b.settings,
			registry:        *NewOrderRegistry(),
		}
	)

	// Read config file.
	What(log.Info().Str("path", filePath), "loading config file...")

	if err := keep.Config.ReadConfigFromFile(filePath, b.settings.EnableDryRun); err != nil {
		return nil, err
	}

	// Optionally augment config.
	if b.augment != nil {
		if err := b.augment(&keep.Config); err != nil {
			return keep, err
		}
	}

	// Assign custom exchange builder.
	keep.ExchangeManager.Builder = b.factory

	// Once everything is set, create and setup exchanges.
	if err := keep.setupExchanges(GCTLog{nil}); err != nil {
		return keep, err
	}

	return keep, nil
}

// +------+
// | Keep |
// +------+

var ErrOrdersAlreadyExists = errors.New("order already exists")

type Keep struct {
	Config          config.Config
	ExchangeManager engine.ExchangeManager
	Root            RootStrategy
	Settings        engine.Settings
	registry        OrderRegistry
}

func (bot *Keep) Run() {
	var wg sync.WaitGroup

	for _, x := range bot.ExchangeManager.GetExchanges() {
		wg.Add(1)

		go func(x exchange.IBotExchange) {
			defer wg.Done()

			err := Stream(bot, x, &bot.Root)

			// This function is never expected to return.  I'm panic()king
			// just to maintain the invariant.
			panic(err)
		}(x)
	}

	wg.Wait()
}

func (bot *Keep) GetOrderValue(exchangeName, orderID string) (OrderValue, bool) {
	return bot.registry.GetOrderValue(exchangeName, orderID)
}

func (bot *Keep) getExchange(x interface{}) exchange.IBotExchange {
	switch x := x.(type) {
	case exchange.IBotExchange:
		return x
	case string:
		e, err := bot.ExchangeManager.GetExchangeByName(x)
		if err != nil {
			panic(fmt.Sprintf("unable to find %s exchange", x))
		}

		return e
	default:
		panic("exchangeOrName should be either an instance of exchange.IBotExchange or a string")
	}
}

// +-------------------------+
// | Keep: Exchange & orders |
// +-------------------------+

func (bot *Keep) GetActiveOrders(exchangeOrName interface{}, request order.GetOrdersRequest) (
	[]order.Detail, error,
) {
	return bot.getExchange(exchangeOrName).GetActiveOrders(&request)
}

func (bot *Keep) SubmitOrder(exchangeOrName interface{}, submit order.Submit) (order.SubmitResponse, error) {
	return bot.SubmitOrderUD(exchangeOrName, submit, nil)
}

func (bot *Keep) SubmitOrderUD(exchangeOrName interface{}, submit order.Submit, userData interface{}) (
	order.SubmitResponse, error,
) {
	e := bot.getExchange(exchangeOrName)

	// Make sure order.Submit.Exchange is properly populated.
	if submit.Exchange == "" {
		submit.Exchange = e.GetName()
	}

	resp, err := e.SubmitOrder(&submit)
	if err == nil {
		if !bot.registry.Store(e.GetName(), resp, userData) {
			return resp, ErrOrdersAlreadyExists
		}
	}

	return resp, err
}

func (bot *Keep) SubmitOrders(e exchange.IBotExchange, xs ...order.Submit) error {
	var wg ErrorWaitGroup

	for _, x := range xs {
		wg.Add(1)

		go func(x order.Submit) {
			_, err := bot.SubmitOrder(e, x)
			wg.Done(err)
		}(x)
	}

	return wg.Wait()
}

func (bot *Keep) CancelAllOrders(exchangeOrName interface{}, assetType asset.Item, pair currency.Pair) (
	order.CancelAllResponse, error,
) {
	e := bot.getExchange(exchangeOrName)

	var cancel order.Cancel
	cancel.Exchange = e.GetName()
	cancel.AssetType = assetType
	cancel.Pair = pair
	cancel.Symbol = pair.String()

	return e.CancelAllOrders(&cancel)
}

func (bot *Keep) CancelOrder(exchangeOrName interface{}, x order.Cancel) error {
	e := bot.getExchange(exchangeOrName)

	if x.Exchange == "" {
		x.Exchange = e.GetName()
	}

	return e.CancelOrder(&x)
}

func (bot *Keep) CancelOrdersByPrefix(exchangeOrName interface{}, x order.Cancel, prefix string) error {
	request := order.GetOrdersRequest{
		Type:      x.Type,
		Side:      x.Side,
		StartTime: time.Time{},
		EndTime:   time.Time{},
		OrderID:   "",
		Pairs:     []currency.Pair{x.Pair},
		AssetType: x.AssetType,
	}

	xs, err := bot.GetActiveOrders(exchangeOrName, request)
	if err != nil {
		return err
	}

	var multi error

	for _, x := range xs {
		// Date is left empty on purpose just to make sure no matching by date is
		// performed.  All the rest is populated as we don't really know which
		// exchange expects what.
		err := bot.CancelOrder(exchangeOrName, order.Cancel{
			Price:         x.Price,
			Amount:        x.Amount,
			Exchange:      bot.getExchange(exchangeOrName).GetName(),
			ID:            x.ID,
			ClientOrderID: x.ClientOrderID,
			AccountID:     x.AccountID,
			ClientID:      x.ClientID,
			WalletAddress: x.WalletAddress,
			Type:          x.Type,
			Side:          x.Side,
			Status:        x.Status,
			AssetType:     x.AssetType,
			Date:          time.Time{},
			Pair:          x.Pair,
			Symbol:        x.Pair.String(),
			Trades:        []order.TradeHistory{},
		})

		multi = multierr.Append(multi, err)
	}

	return multi
}

// +-------------------------+
// | Keep: Event observation |
// +-------------------------+

func (bot *Keep) OnOrder(e exchange.IBotExchange, x order.Detail) {
	if x.Status == order.Filled {
		value, ok := bot.GetOrderValue(e.GetName(), x.ID)
		if !ok {
			// No user data for this order.
			return
		}

		if obs, ok := value.UserData.(OnFilledObserver); ok {
			obs.OnFilled(bot, e, x)
		}
	}
}

// +-------------------------------+
// | Keep: GCT compatibility layer |
// +-------------------------------+

type GCTLog struct {
	ExchangeSys interface{}
}

func (g GCTLog) Warnf(_ interface{}, data string, v ...interface{}) {
	What(log.Warn(), fmt.Sprintf(data, v...))
}

func (g GCTLog) Errorf(_ interface{}, data string, v ...interface{}) {
	What(log.Error(), fmt.Sprintf(data, v...))
}

func (g GCTLog) Debugf(_ interface{}, data string, v ...interface{}) {
	What(log.Debug(), fmt.Sprintf(data, v...))
}

func (bot *Keep) LoadExchange(name string, wg *sync.WaitGroup) error {
	return bot.loadExchange(name, wg, GCTLog{nil})
}

// +----------------------------+
// | Copied from gocryptotrader |
// +----------------------------+

var (
	ErrNoExchangesLoaded    = errors.New("no exchanges have been loaded")
	ErrExchangeFailedToLoad = errors.New("exchange failed to load")
)

// loadExchange is an unchanged copy of Engine.LoadExchange.
//
// nolint
func (bot *Keep) loadExchange(name string, wg *sync.WaitGroup, gctlog GCTLog) error {
	exch, err := bot.ExchangeManager.NewExchangeByName(name)
	if err != nil {
		return err
	}
	if exch.GetBase() == nil {
		return ErrExchangeFailedToLoad
	}

	var localWG sync.WaitGroup
	localWG.Add(1)
	go func() {
		exch.SetDefaults()
		localWG.Done()
	}()
	exchCfg, err := bot.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	if bot.Settings.EnableAllPairs &&
		exchCfg.CurrencyPairs != nil {
		assets := exchCfg.CurrencyPairs.GetAssetTypes(false)
		for x := range assets {
			var pairs currency.Pairs
			pairs, err = exchCfg.CurrencyPairs.GetPairs(assets[x], false)
			if err != nil {
				return err
			}
			exchCfg.CurrencyPairs.StorePairs(assets[x], pairs, true)
		}
	}

	if bot.Settings.EnableExchangeVerbose {
		exchCfg.Verbose = true
	}
	if exchCfg.Features != nil {
		if bot.Settings.EnableExchangeWebsocketSupport &&
			exchCfg.Features.Supports.Websocket {
			exchCfg.Features.Enabled.Websocket = true
		}
		if bot.Settings.EnableExchangeAutoPairUpdates &&
			exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
			exchCfg.Features.Enabled.AutoPairUpdates = true
		}
		if bot.Settings.DisableExchangeAutoPairUpdates {
			if exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
				exchCfg.Features.Enabled.AutoPairUpdates = false
			}
		}
	}
	if bot.Settings.HTTPUserAgent != "" {
		exchCfg.HTTPUserAgent = bot.Settings.HTTPUserAgent
	}
	if bot.Settings.HTTPProxy != "" {
		exchCfg.ProxyAddress = bot.Settings.HTTPProxy
	}
	if bot.Settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		exchCfg.HTTPTimeout = bot.Settings.HTTPTimeout
	}
	if bot.Settings.EnableExchangeHTTPDebugging {
		exchCfg.HTTPDebugging = bot.Settings.EnableExchangeHTTPDebugging
	}

	localWG.Wait()
	if !bot.Settings.EnableExchangeHTTPRateLimiter {
		gctlog.Warnf(gctlog.ExchangeSys,
			"Loaded exchange %s rate limiting has been turned off.\n",
			exch.GetName(),
		)
		err = exch.DisableRateLimiter()
		if err != nil {
			gctlog.Errorf(gctlog.ExchangeSys,
				"Loaded exchange %s rate limiting cannot be turned off: %s.\n",
				exch.GetName(),
				err,
			)
		}
	}

	exchCfg.Enabled = true
	err = exch.Setup(exchCfg)
	if err != nil {
		exchCfg.Enabled = false
		return err
	}

	bot.ExchangeManager.Add(exch)
	base := exch.GetBase()
	if base.API.AuthenticatedSupport ||
		base.API.AuthenticatedWebsocketSupport {
		assetTypes := base.GetAssetTypes(false)
		var useAsset asset.Item
		for a := range assetTypes {
			err = base.CurrencyPairs.IsAssetEnabled(assetTypes[a])
			if err != nil {
				continue
			}
			useAsset = assetTypes[a]
			break
		}
		err = exch.ValidateCredentials(useAsset)
		if err != nil {
			gctlog.Warnf(gctlog.ExchangeSys,
				"%s: Cannot validate credentials, authenticated support has been disabled, Error: %s\n",
				base.Name,
				err)
			base.API.AuthenticatedSupport = false
			base.API.AuthenticatedWebsocketSupport = false
			exchCfg.API.AuthenticatedSupport = false
			exchCfg.API.AuthenticatedWebsocketSupport = false
		}
	}

	if wg != nil {
		exch.Start(wg)
	} else {
		tempWG := sync.WaitGroup{}
		exch.Start(&tempWG)
		tempWG.Wait()
	}

	return nil
}

// setupExchanges is an (almost) unchanged copy of Engine.SetupExchanges.
//
// nolint
func (bot *Keep) setupExchanges(gctlog GCTLog) error {
	var wg sync.WaitGroup
	configs := bot.Config.GetAllExchangeConfigs()

	// DELETED: parameters -> dryRun...()

	for x := range configs {
		if !configs[x].Enabled && !bot.Settings.EnableAllExchanges {
			gctlog.Debugf(gctlog.ExchangeSys, "%s: Exchange support: Disabled\n", configs[x].Name)
			continue
		}
		wg.Add(1)
		go func(c config.ExchangeConfig) {
			defer wg.Done()
			err := bot.LoadExchange(c.Name, &wg)
			if err != nil {
				gctlog.Errorf(gctlog.ExchangeSys, "LoadExchange %s failed: %s\n", c.Name, err)
				return
			}
			gctlog.Debugf(gctlog.ExchangeSys,
				"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
				c.Name,
				common.IsEnabled(c.API.AuthenticatedSupport),
				common.IsEnabled(c.Verbose),
			)
		}(configs[x])
	}
	wg.Wait()
	if len(bot.ExchangeManager.GetExchanges()) == 0 {
		return ErrNoExchangesLoaded
	}
	return nil
}
