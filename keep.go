package dola

import (
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

func NewKeep(settings engine.Settings) (*Keep, error) {
	if settings.ConfigFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return &Keep{}, err
		}
		path := filepath.Join(home, ".dola/config.json")
		settings.ConfigFile = path
	}
	e := &Keep{
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

	f := func(x exchange.IBotExchange) {
		defer wg.Done()

		err := Stream(k, x, &k.Root)
		// This function is never expected to return.  I'm panic()king
		// just to maintain the invariant.
		panic(err)
	}

	for _, x := range k.exchangeManager.GetExchanges() {
		wg.Add(1)
		go f(x)
	}
	wg.Wait()
}

func (k *Keep) exchangeConfig(name string) (*config.ExchangeConfig, error) {
	conf, err := k.config.GetExchangeConfig(name)
	if err != nil {
		return conf, err
	}

	if k.settings.EnableAllPairs &&
		conf.CurrencyPairs != nil {
		assets := conf.CurrencyPairs.GetAssetTypes(false)
		for x := range assets {
			var pairs currency.Pairs
			pairs, err = conf.CurrencyPairs.GetPairs(assets[x], false)
			if err != nil {
				return conf, err
			}
			conf.CurrencyPairs.StorePairs(assets[x], pairs, true)
		}
	}

	if k.settings.EnableExchangeVerbose {
		conf.Verbose = true
	}
	if conf.Features != nil {
		if k.settings.EnableExchangeWebsocketSupport && conf.Features.Supports.Websocket {
			conf.Features.Enabled.Websocket = true
		}
		if k.settings.EnableExchangeAutoPairUpdates && conf.Features.Supports.RESTCapabilities.AutoPairUpdates {
			conf.Features.Enabled.AutoPairUpdates = true
		}
		if k.settings.DisableExchangeAutoPairUpdates && conf.Features.Supports.RESTCapabilities.AutoPairUpdates {
			conf.Features.Enabled.AutoPairUpdates = false
		}
	}
	if k.settings.HTTPUserAgent != "" {
		conf.HTTPUserAgent = k.settings.HTTPUserAgent
	}
	if k.settings.HTTPProxy != "" {
		conf.ProxyAddress = k.settings.HTTPProxy
	}
	if k.settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		conf.HTTPTimeout = k.settings.HTTPTimeout
	}
	if k.settings.EnableExchangeHTTPDebugging {
		conf.HTTPDebugging = k.settings.EnableExchangeHTTPDebugging
	}

	return conf, nil
}

// func (k *Keep) Subscribe(x exchange.IBotExchange, f Subscriber) {
// 	k.subs[x.GetName()].Add(f)
// }

// loadExchange is copied from github.com/thrasher-corp/gocryptotrader.
func (k *Keep) loadExchange(name string, wg *sync.WaitGroup) error {
	exch, err := k.exchangeManager.NewExchangeByName(name)
	if err != nil {
		return err
	}
	base := exch.GetBase()
	if base == nil {
		panic("invalid state")
	}

	// This, it looks like, is a lengthy operation.  In the original code (of
	// gocryptotrader) it was moved in a goroutine.
	exch.SetDefaults()

	conf, err := k.exchangeConfig(exch.GetName())
	if err != nil {
		return err
	}

	if !k.settings.EnableExchangeHTTPRateLimiter {
		log.Info().
			Str("what", "Loaded exchange rate limiting has been turned off.").
			Str("exchange", name).
			Msg(Location())
		err = exch.DisableRateLimiter()
		if err != nil {
			log.Error().
				Err(err).
				Str("what", "Loaded exchange %s rate limiting cannot be turned off.").
				Str("exchange", name).
				Msg(Location())
		}
	}

	if err := exch.Setup(conf); err != nil {
		conf.Enabled = false

		return err
	}

	conf.Enabled = true
	k.exchangeManager.Add(exch)

	if base.API.AuthenticatedSupport || base.API.AuthenticatedWebsocketSupport {
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
			conf.API.AuthenticatedSupport = false
			conf.API.AuthenticatedWebsocketSupport = false
		}
	}

	exch.Start(wg)

	return nil
}

func (k *Keep) setupExchanges() {
	var wg sync.WaitGroup

	f := func(c config.ExchangeConfig) {
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
	}

	configs := k.config.GetAllExchangeConfigs()
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
		go f(configs[x])
	}
	wg.Wait()
	// for _, x := range k.exchangeManager.GetExchanges() {
	// 	k.subs[x.GetName()] = &Multiplexer{}
	// }
}
