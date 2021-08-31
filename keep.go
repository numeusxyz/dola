package dola

import (
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var ErrOrdersAlreadyExists = errors.New("order already exists")

type Keep struct {
	Root            RootStrategy
	Settings        engine.Settings
	Config          config.Config
	ExchangeManager engine.ExchangeManager
	registry        OrderRegistry
}

func NewKeep(settings engine.Settings) *Keep {
	settings.ConfigFile = ConfigFile(settings.ConfigFile)

	var conf config.Config

	return &Keep{
		Root:            NewRootStrategy(),
		Settings:        settings,
		Config:          conf,
		ExchangeManager: *engine.SetupExchangeManager(),
		registry:        *NewOrderRegistry(),
	}
}

func (bot *Keep) Setup() error {
	filePath, err := config.GetAndMigrateDefaultPath(bot.Settings.ConfigFile)
	if err != nil {
		return err
	}

	What(log.Info().Str("path", filePath), "loading config file...")

	if err := bot.Config.ReadConfigFromFile(filePath, bot.Settings.EnableDryRun); err != nil {
		return err
	}

	if err := bot.setupExchanges(GCTLog{nil}); err != nil {
		return err
	}

	return nil
}

func (bot *Keep) SubmitOrder(exchangeOrName interface{}, submit order.Submit) (order.SubmitResponse, error) {
	return bot.SubmitOrderUD(exchangeOrName, submit, nil)
}

func (bot *Keep) SubmitOrderUD(exchangeOrName interface{}, submit order.Submit, userData interface{}) (
	order.SubmitResponse, error,
) {
	e := bot.getExchange(exchangeOrName)

	// Make sure order.Submit.Exchange is properly populated.
	submit.Exchange = e.GetName()

	// Do we want to generate a custom order ID in case x.ClientOrderID is empty?

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

func (bot *Keep) CancelOrder(exchangeOrName interface{}, x order.Cancel) error {
	e := bot.getExchange(exchangeOrName)

	return e.CancelOrder(&x)
}

func (bot *Keep) CancelAllOrders(exchangeOrName interface{}, assetType asset.Item, pair currency.Pair) (
	order.CancelAllResponse, error,
) {
	e := bot.getExchange(exchangeOrName)

	var cancel order.Cancel
	cancel.Exchange = e.GetName()
	cancel.AssetType = assetType
	cancel.Pair = pair
	// cancel.Symbol = pair.String()

	return e.CancelAllOrders(&cancel)
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

// +-------------------+
// | Event observation |
// +-------------------+

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

// +-------------------------+
// | GCT compatibility layer |
// +-------------------------+

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

func (bot *Keep) LoadExchange(c *config.ExchangeConfig, wg *sync.WaitGroup) error {
	return bot.loadExchange(c, wg, GCTLog{nil})
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
func (bot *Keep) loadExchange(c *config.ExchangeConfig, wg *sync.WaitGroup, gctlog GCTLog) error {
	exch, err := bot.ExchangeManager.NewExchangeByName(c.Name)
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

	if bot.Settings.EnableAllPairs &&
		c.CurrencyPairs != nil {
		assets := c.CurrencyPairs.GetAssetTypes(false)
		for x := range assets {
			var pairs currency.Pairs
			pairs, err = c.CurrencyPairs.GetPairs(assets[x], false)
			if err != nil {
				return err
			}
			c.CurrencyPairs.StorePairs(assets[x], pairs, true)
		}
	}

	if bot.Settings.EnableExchangeVerbose {
		c.Verbose = true
	}
	if c.Features != nil {
		if bot.Settings.EnableExchangeWebsocketSupport &&
			c.Features.Supports.Websocket {
			c.Features.Enabled.Websocket = true
		}
		if bot.Settings.EnableExchangeAutoPairUpdates &&
			c.Features.Supports.RESTCapabilities.AutoPairUpdates {
			c.Features.Enabled.AutoPairUpdates = true
		}
		if bot.Settings.DisableExchangeAutoPairUpdates {
			if c.Features.Supports.RESTCapabilities.AutoPairUpdates {
				c.Features.Enabled.AutoPairUpdates = false
			}
		}
	}
	if bot.Settings.HTTPUserAgent != "" {
		c.HTTPUserAgent = bot.Settings.HTTPUserAgent
	}
	if bot.Settings.HTTPProxy != "" {
		c.ProxyAddress = bot.Settings.HTTPProxy
	}
	if bot.Settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		c.HTTPTimeout = bot.Settings.HTTPTimeout
	}
	if bot.Settings.EnableExchangeHTTPDebugging {
		c.HTTPDebugging = bot.Settings.EnableExchangeHTTPDebugging
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

	c.Enabled = true
	err = exch.Setup(c)
	if err != nil {
		c.Enabled = false
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
			c.API.AuthenticatedSupport = false
			c.API.AuthenticatedWebsocketSupport = false
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
			err := bot.LoadExchange(&c, &wg)
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
