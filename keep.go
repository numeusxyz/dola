package dola

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Keep struct {
	Root RootStrategy

	Settings        engine.Settings
	Config          config.Config
	ExchangeManager engine.ExchangeManager
}

func NewKeep(settings engine.Settings) (*Keep, error) {
	if settings.ConfigFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return &Keep{}, err
		}

		path := filepath.Join(home, ".dola/config.json")
		settings.ConfigFile = path
	}

	keep := &Keep{
		Settings:        settings,
		ExchangeManager: *engine.SetupExchangeManager(),
	}

	filePath, err := config.GetAndMigrateDefaultPath(keep.Settings.ConfigFile)
	if err != nil {
		return keep, err
	}

	if err := keep.Config.ReadConfigFromFile(filePath, keep.Settings.EnableDryRun); err != nil {
		return keep, err
	}

	if err := keep.setupExchanges(GCTLog{}); err != nil {
		return keep, err
	}

	return keep, nil
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

// +-------------------------+
// | GCT compatibility layer |
// +-------------------------+

type GCTLog struct {
	ExchangeSys struct{}
}

func (g GCTLog) Warnf(_ interface{}, data string, v ...interface{}) {
	log.Warn().Str("what", fmt.Sprintf(data, v...)).Msg(Location())
}

func (g GCTLog) Errorf(_ interface{}, data string, v ...interface{}) {
	log.Error().Str("what", fmt.Sprintf(data, v...)).Msg(Location())
}

func (g GCTLog) Debugf(_ interface{}, data string, v ...interface{}) {
	log.Debug().Str("what", fmt.Sprintf(data, v...)).Msg(Location())
}

func (bot *Keep) LoadExchange(name string, wg *sync.WaitGroup) error {
	return bot.loadExchange(name, wg, GCTLog{})
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
