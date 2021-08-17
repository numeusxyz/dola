package dola

import (
	"errors"
	"fmt"
	"os"
	"strings"
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
)

var ErrOrdersAlreadyExists = errors.New("order already exists")

type Keep struct {
	Root RootStrategy

	Settings        engine.Settings
	Config          config.Config
	ExchangeManager engine.ExchangeManager
	Registry        OrderRegistry
}

func NewKeep(settings engine.Settings) (*Keep, error) {
	settings.ConfigFile = configFile(settings.ConfigFile)

	keep := &Keep{
		Root:            NewRootStrategy(),
		Settings:        settings,
		Config:          config.Config{}, // nolint: exhaustivestruct
		ExchangeManager: *engine.SetupExchangeManager(),
		Registry:        NewOrderRegistry(),
	}

	filePath, err := config.GetAndMigrateDefaultPath(keep.Settings.ConfigFile)
	if err != nil {
		return keep, err
	}

	Msg(log.Info().Str("path", filePath), "loading config file...", "")

	if err := keep.Config.ReadConfigFromFile(filePath, keep.Settings.EnableDryRun); err != nil {
		return keep, err
	}

	if err := keep.setupExchanges(GCTLog{nil}); err != nil {
		return keep, err
	}

	return keep, nil
}

func (bot *Keep) SubmitOrder(exchangeOrName interface{}, submit order.Submit) (order.SubmitResponse, error) {
	return bot.SubmitOrderUD(exchangeOrName, submit, nil)
}

func (bot *Keep) SubmitOrderUD(
	exchangeOrName interface{},
	submit order.Submit,
	userData interface{},
) (order.SubmitResponse, error) {
	e := bot.getExchange(exchangeOrName)

	// Make sure order.Submit.Exchange is properly populated.
	submit.Exchange = e.GetName()

	// Do we want to generate a custom order ID in case x.ClientOrderID is empty?

	resp, err := e.SubmitOrder(&submit)
	if err == nil {
		if !bot.Registry.OnSubmit(e.GetName(), resp, userData) {
			return resp, ErrOrdersAlreadyExists
		}
	}

	return resp, err
}

func (bot *Keep) SubmitOrders(e exchange.IBotExchange, xs ...order.Submit) error {
	group := sync.WaitGroup{}
	multi := NewMultiErr(nil)

	for _, x := range xs {
		group.Add(1)

		go func(x order.Submit) {
			defer group.Done()

			_, err := bot.SubmitOrder(e, x)
			multi.Append(err)
		}(x)
	}

	group.Done()

	return multi.Err()
}

func (bot *Keep) CancelOrder(e exchange.IBotExchange, x order.Cancel) error {
	return e.CancelOrder(&x)
}

func (bot *Keep) CancelAllOrders(
	e exchange.IBotExchange, assetType asset.Item, pair currency.Pair,
) (order.CancelAllResponse, error) {
	return e.CancelAllOrders(&order.Cancel{
		Price:         0,
		Amount:        0,
		Exchange:      e.GetName(),
		ID:            "",
		ClientOrderID: "",
		AccountID:     "",
		ClientID:      "",
		WalletAddress: "",
		Type:          "",
		Side:          "",
		Status:        "",
		AssetType:     assetType,
		Date:          time.Time{},
		Pair:          pair,
		Symbol:        "",
		Trades:        []order.TradeHistory{},
	})
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
	return bot.Registry.GetOrderValue(exchangeName, orderID)
}

func (bot *Keep) getExchange(x interface{}) exchange.IBotExchange {
	switch x := x.(type) {
	case exchange.IBotExchange:
		return x
	case string:
		return bot.ExchangeManager.GetExchangeByName(x)
	default:
		panic("exchangeOrName should be either an instance of exchange.IBotExchange or a string")
	}
}

func configFile(inp string) string {
	if inp != "" {
		path := expandUser(inp)
		if fileExists(path) {
			return path
		}
	}

	if env := os.Getenv("DOLA_CONFIG"); env != "" {
		path := expandUser(env)
		if fileExists(path) {
			return path
		}
	}

	if path := expandUser("~/.dola/config.json"); fileExists(path) {
		return path
	}

	return ""
}

func expandUser(path string) string {
	return os.ExpandEnv(strings.Replace(path, "~", "$HOME", 1))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return !os.IsNotExist(err)
}

// +-------------------------+
// | GCT compatibility layer |
// +-------------------------+

type GCTLog struct {
	ExchangeSys interface{}
}

func (g GCTLog) Warnf(_ interface{}, data string, v ...interface{}) {
	Msg(log.Warn(), fmt.Sprintf(data, v...), "")
}

func (g GCTLog) Errorf(_ interface{}, data string, v ...interface{}) {
	Msg(log.Error(), fmt.Sprintf(data, v...), "")
}

func (g GCTLog) Debugf(_ interface{}, data string, v ...interface{}) {
	Msg(log.Debug(), fmt.Sprintf(data, v...), "")
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

// setupExchanges is an unchanged copy of Engine.SetupExchanges.
//
// nolint
func (bot *Keep) setupExchanges(gctlog GCTLog) error {
	var wg sync.WaitGroup
	configs := bot.Config.GetAllExchangeConfigs()
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
