package dola

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Keep struct {
	Root RootStrategy

	settings        engine.Settings
	config          config.Config
	exchangeManager engine.ExchangeManager
	// subs            map[string]*Multiplexer
}

func NewKeep(settings engine.Settings) (Keep, error) {
	if settings.ConfigFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Keep{}, err
		}
		path := filepath.Join(home, ".dola/config.json")
		settings.ConfigFile = path
	}
	e := Keep{
		settings:        settings,
		exchangeManager: *engine.SetupExchangeManager(),
		// subs:            make(map[string]*Multiplexer),
	}

	filePath, err := config.GetAndMigrateDefaultPath(e.settings.ConfigFile)
	if err != nil {
		return e, err
	}

	err = e.config.ReadConfigFromFile(filePath, e.settings.EnableDryRun)
	if err != nil {
		return e, err
	}

	e.setupExchanges()
	return e, nil
}

func (k *Keep) Run() {
	var wg sync.WaitGroup
	for _, x := range k.exchangeManager.GetExchanges() {
		wg.Add(1)
		go func(x exchange.IBotExchange) {
			defer wg.Done()

			err := Stream(k, x, &k.Root)
			// This function is never expected to return.  I'm panic()king
			// just to maintain the invariant.
			panic(err)
		}(x)
	}
	wg.Wait()
}

// func (k *Keep) Subscribe(x exchange.IBotExchange, f Subscriber) {
// 	k.subs[x.GetName()].Add(f)
// }

func (k *Keep) loadExchange(name string, wg *sync.WaitGroup) error {
	exch, err := k.exchangeManager.NewExchangeByName(name)
	if err != nil {
		return err
	}
	if exch.GetBase() == nil {
		return errors.New("TODO")
	}

	var localWG sync.WaitGroup
	localWG.Add(1)
	go func() {
		CheckerPush()
		defer CheckerPop()

		exch.SetDefaults()
		localWG.Done()
	}()
	exchCfg, err := k.config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	if k.settings.EnableAllPairs &&
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

	if k.settings.EnableExchangeVerbose {
		exchCfg.Verbose = true
	}
	if exchCfg.Features != nil {
		if k.settings.EnableExchangeWebsocketSupport && exchCfg.Features.Supports.Websocket {
			exchCfg.Features.Enabled.Websocket = true
		}
		if k.settings.EnableExchangeAutoPairUpdates && exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
			exchCfg.Features.Enabled.AutoPairUpdates = true
		}
		if k.settings.DisableExchangeAutoPairUpdates && exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
			exchCfg.Features.Enabled.AutoPairUpdates = false
		}
	}
	if k.settings.HTTPUserAgent != "" {
		exchCfg.HTTPUserAgent = k.settings.HTTPUserAgent
	}
	if k.settings.HTTPProxy != "" {
		exchCfg.ProxyAddress = k.settings.HTTPProxy
	}
	if k.settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		exchCfg.HTTPTimeout = k.settings.HTTPTimeout
	}
	if k.settings.EnableExchangeHTTPDebugging {
		exchCfg.HTTPDebugging = k.settings.EnableExchangeHTTPDebugging
	}

	localWG.Wait()
	if !k.settings.EnableExchangeHTTPRateLimiter {
		log.Info().
			Str("what", "Loaded exchange rate limiting has been turned off.").
			Str("exchange", exch.GetName()).
			Msg(Location())
		err = exch.DisableRateLimiter()
		if err != nil {
			log.Error().
				Err(err).
				Str("what", "Loaded exchange %s rate limiting cannot be turned off.").
				Str("exchange", exch.GetName()).
				Msg(Location())
		}
	}

	exchCfg.Enabled = true
	err = exch.Setup(exchCfg)
	if err != nil {
		exchCfg.Enabled = false
		return err
	}

	k.exchangeManager.Add(exch)
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
			log.Warn().
				Str("what", "Cannot validate credentials, authenticated support has been disabled.").
				Str("base.Name", base.Name).
				Err(err).
				Msg(Location())
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

func (k *Keep) setupExchanges() {
	configs := k.config.GetAllExchangeConfigs()

	var wg sync.WaitGroup
	for x := range configs {
		if !configs[x].Enabled && !k.settings.EnableAllExchanges {
			if e := log.Debug(); e.Enabled() {
				e.Str("what", "Exchange support: Disabled").
					Str("exchange", configs[x].Name).
					Msg(Location())
			}
			continue
		}
		wg.Add(1)
		go func(c config.ExchangeConfig) {
			defer wg.Done()
			err := k.loadExchange(c.Name, &wg)
			if err != nil {
				log.Error().
					Str("what", "LoadExchange() failed.").
					Str("exchange", c.Name).
					Err(err).
					Msg(Location())
			}
			if e := log.Debug(); e.Enabled() {
				e.Str("what", "Exchange support: Enabled.").
					Str("exchange", c.Name).
					Bool("AuthenticatedSupport", c.API.AuthenticatedSupport).
					Bool("Verbose", c.Verbose).
					Msg(Location())
			}
		}(configs[x])
	}
	wg.Wait()
	// for _, x := range k.exchangeManager.GetExchanges() {
	// 	k.subs[x.GetName()] = &Multiplexer{}
	// }
}
