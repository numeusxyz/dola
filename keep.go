package dola

import (
	"context"
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
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"go.uber.org/multierr"
)

var ErrNoAssetType = errors.New("asset type not associated with currency pair")

const (
	defaultWebsocketTrafficTimeout = time.Second * 30
)

// +-------------+
// | KeepBuilder |
// +-------------+

type (
	AugmentConfigFunc func(*config.Config) error
)

type KeepBuilder struct {
	augment             AugmentConfigFunc
	balancesRefreshRate time.Duration
	factory             ExchangeFactory
	settings            engine.Settings
	reporters           []Reporter
}

func NewKeepBuilder() *KeepBuilder {
	var settings engine.Settings

	return &KeepBuilder{
		augment:             nil,
		balancesRefreshRate: 0,
		factory:             nil,
		settings:            settings,
		reporters:           []Reporter{},
	}
}

func (b *KeepBuilder) Augment(f AugmentConfigFunc) *KeepBuilder {
	b.augment = f

	return b
}

func (b *KeepBuilder) Balances(refreshRate time.Duration) *KeepBuilder {
	b.balancesRefreshRate = refreshRate

	return b
}

func (b *KeepBuilder) CustomExchange(f ExchangeFactory) *KeepBuilder {
	b.factory = f

	return b
}

func (b *KeepBuilder) Settings(s engine.Settings) *KeepBuilder {
	b.settings = s

	return b
}

func (b *KeepBuilder) Reporter(r Reporter) *KeepBuilder {
	b.reporters = append(b.reporters, r)

	return b
}

// nolint: funlen
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
			reporters:       b.reporters,
		}
	)

	// Add history strategy: a special type of strategy that may keep multiple
	// channels of historical data.
	hist := NewHistoryStrategy()
	keep.Root.Add("history", &hist)

	// Optionally add the balances strategy that keeps track of available balances per
	// exchange.
	if b.balancesRefreshRate > 0 {
		keep.Root.Add("balances", NewBalancesStrategy(b.balancesRefreshRate))
	}

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

	// enable GCT's verbose output through our logging system
	if b.settings.Verbose {
		keep.setupGCTLogging()

		gctlog.Infoln(gctlog.Global, "GCT logger initialised.")
	}

	// Once everything is set, create and setup exchanges.
	if err := keep.setupExchanges(); err != nil {
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
	reporters       []Reporter
}

// Run is the entry point of all exchange data streams.  Strategy.On*() events for a
// single exchange are invoked from the same thread.  Thus, if a strategy deals with
// multiple exchanges simultaneously, there may be race conditions.
func (bot *Keep) Run(ctx context.Context) {
	var wg sync.WaitGroup

	exchgs, err := bot.ExchangeManager.GetExchanges()
	if err != nil {
		panic(err)
	}

	for _, x := range exchgs {
		wg.Add(1)

		go func(x exchange.IBotExchange) {
			defer wg.Done()

			// fetch the root strategy
			s := &bot.Root

			// Init root strategy for this exchange.
			if err := s.Init(ctx, bot, x); err != nil {
				panic(fmt.Errorf("failed to initialize strategy: %w", err))
			}

			// go into an infinite loop, either handling websocket
			// events or just plain blocked when there are none
			err := Loop(ctx, bot, x, s)
			// nolint: godox
			// TODO: handle err on terminate when context gets cancelled

			// Deinit root strategy for this exchange.
			if err := s.Deinit(bot, x); err != nil {
				panic(err)
			}

			// This function is never expected to return.  I'm panic()king
			// just to maintain the invariant.
			panic(err)
		}(x)
	}

	wg.Wait()
}

func Loop(ctx context.Context, k *Keep, e exchange.IBotExchange, s Strategy) error {
	// If this exchange doesn't support websockets we still need to
	// keep running
	if !e.IsWebsocketEnabled() {
		What(log.Warn().Str("exchange", e.GetName()), "no websocket support")

		<-ctx.Done()

		return nil
	}

	// this exchanges does support websockets, go into an
	// infinite loop of receiving/handling messages
	return Stream(ctx, k, e, s)
}

func (bot *Keep) AddHistorian(
	exchangeName,
	eventName string,
	interval time.Duration,
	stateLength int,
	f func(Array),
) error {
	strategy, err := bot.Root.Get("history")
	if err != nil {
		return err
	}

	hist, ok := strategy.(*HistoryStrategy)
	if !ok {
		panic("")
	}

	return hist.AddHistorian(exchangeName, eventName, interval, stateLength, f)
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

// +----------------------+
// | Keep: Exchange state |
// +----------------------+

func (bot *Keep) GetActiveOrders(ctx context.Context, exchangeOrName interface{}, request order.GetOrdersRequest) (
	[]order.Detail, error,
) {
	e := bot.getExchange(exchangeOrName)

	bot.ReportEvent(GetActiveOrdersMetric, e.GetName())

	timer := time.Now()

	defer bot.ReportLatency(GetActiveOrdersLatencyMetric, timer, e.GetName())

	resp, err := e.GetActiveOrders(ctx, &request)
	if err != nil {
		bot.ReportEvent(GetActiveOrdersErrorMetric, e.GetName())

		return resp, err
	}

	return resp, nil
}

// +------------------------+
// | Keep: Order submission |
// +------------------------+

func (bot *Keep) SubmitOrder(ctx context.Context,
	exchangeOrName interface{},
	submit order.Submit) (order.SubmitResponse, error) {
	return bot.SubmitOrderUD(ctx, exchangeOrName, submit, nil)
}

func (bot *Keep) SubmitOrderUD(ctx context.Context,
	exchangeOrName interface{},
	submit order.Submit,
	userData interface{}) (
	order.SubmitResponse, error,
) {
	e := bot.getExchange(exchangeOrName)

	// Make sure order.Submit.Exchange is properly populated.
	if submit.Exchange == "" {
		submit.Exchange = e.GetName()
	}

	bot.ReportEvent(SubmitOrderMetric, e.GetName())

	defer bot.ReportLatency(SubmitOrderLatencyMetric, time.Now(), e.GetName())

	resp, err := e.SubmitOrder(ctx, &submit)
	if err != nil {
		// post an error metric event
		bot.ReportEvent(SubmitOrderErrorMetric, e.GetName())

		return resp, err
	}

	// store the order in the registry
	if !bot.registry.Store(e.GetName(), resp, userData) {
		return resp, ErrOrdersAlreadyExists
	}

	return resp, err
}

func (bot *Keep) SubmitOrders(ctx context.Context, e exchange.IBotExchange, xs ...order.Submit) error {
	var wg ErrorWaitGroup

	bot.ReportEvent(SubmitBulkOrderMetric, e.GetName())

	defer bot.ReportLatency(SubmitBulkOrderLatencyMetric, time.Now(), e.GetName())

	for _, x := range xs {
		wg.Add(1)

		go func(x order.Submit) {
			_, err := bot.SubmitOrder(ctx, e, x)
			wg.Done(err)
		}(x)
	}

	return wg.Wait()
}

func (bot *Keep) ModifyOrder(ctx context.Context,
	exchangeOrName interface{},
	mod order.Modify) (order.Modify, error) {
	e := bot.getExchange(exchangeOrName)

	bot.ReportEvent(ModifyOrderMetric, e.GetName())

	defer bot.ReportLatency(ModifyOrderLatencyMetric, time.Now(), e.GetName())

	resp, err := e.ModifyOrder(ctx, &mod)
	if err != nil {
		// post an error metric event
		bot.ReportEvent(ModifyOrderErrorMetric, e.GetName())

		return resp, err
	}

	return resp, nil
}

// +--------------------------+
// | Keep: Order cancellation |
// +--------------------------+

func (bot *Keep) CancelAllOrders(ctx context.Context,
	exchangeOrName interface{},
	assetType asset.Item,
	pair currency.Pair) (
	order.CancelAllResponse, error,
) {
	e := bot.getExchange(exchangeOrName)

	var cancel order.Cancel
	cancel.Exchange = e.GetName()
	cancel.AssetType = assetType
	cancel.Pair = pair
	cancel.Symbol = pair.String()

	bot.ReportEvent(CancelAllOrdersMetric, e.GetName(), pair.String())

	defer bot.ReportLatency(CancelAllOrdersLatencyMetric, time.Now(), e.GetName())

	resp, err := e.CancelAllOrders(ctx, &cancel)
	if err != nil {
		bot.ReportEvent(CancelAllOrdersErrorMetric, e.GetName())

		return resp, err
	}

	return resp, nil
}

func (bot *Keep) CancelOrder(ctx context.Context, exchangeOrName interface{}, x order.Cancel) error {
	e := bot.getExchange(exchangeOrName)

	if x.Exchange == "" {
		x.Exchange = e.GetName()
	}

	bot.ReportEvent(CancelOrderMetric, e.GetName())

	defer bot.ReportLatency(CancelOrderLatencyMetric, time.Now(), e.GetName())

	if err := e.CancelOrder(ctx, &x); err != nil {
		// post an error metric event
		bot.ReportEvent(CancelOrderErrorMetric, e.GetName())

		return err
	}

	return nil
}

func (bot *Keep) CancelOrdersByPrefix(ctx context.Context,
	exchangeOrName interface{},
	x order.Cancel,
	prefix string) error {
	request := order.GetOrdersRequest{
		Type:      x.Type,
		Side:      x.Side,
		StartTime: time.Time{},
		EndTime:   time.Time{},
		OrderID:   "",
		Pairs:     []currency.Pair{x.Pair},
		AssetType: x.AssetType,
	}

	xs, err := bot.GetActiveOrders(ctx, exchangeOrName, request)
	if err != nil {
		return err
	}

	var multi error

	for _, x := range xs {
		// Date is left empty on purpose just to make sure no matching by date is
		// performed.  All the rest is populated as we don't really know which
		// exchange expects what.
		err := bot.CancelOrder(ctx, exchangeOrName, order.Cancel{
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

// +----------------------+
// | Keep: Metric reports |
// +----------------------+

func (bot *Keep) ReportLatency(m Metric, t time.Time, labels ...string) {
	for _, r := range bot.reporters {
		r.Latency(m, time.Since(t), labels...)
	}
}

func (bot *Keep) ReportEvent(m Metric, labels ...string) {
	for _, r := range bot.reporters {
		r.Event(m, labels...)
	}
}

func (bot *Keep) ReportValue(m Metric, v float64, labels ...string) {
	for _, r := range bot.reporters {
		r.Value(m, v, labels...)
	}
}

// +-------------------------------+
// | Keep: GCT wrapped function    |
// +-------------------------------+

// GetExchanges is a wrapper of GCT's Engine.GetExchanges.
//nolint
func (bot *Keep) GetExchanges() []exchange.IBotExchange {
	exchgs, err := bot.ExchangeManager.GetExchanges()
	if err != nil {
		What(log.Warn().
			Err(err), "unable to get exchanges")

		return []exchange.IBotExchange{}
	}

	return exchgs
}

// ActivatePair activates and makes available the provided currency
// pair.
// nolint: stylecheck
func (bot *Keep) ActivateAsset(e exchange.IBotExchange, a asset.Item) error {
	base := e.GetBase()

	// asset type was not previously enabled, take care of that here
	if err := base.CurrencyPairs.SetAssetEnabled(a, true); err != nil && !errors.Is(err, currency.ErrAssetAlreadyEnabled) {
		return err
	}

	return nil
}

// ActivatePair activates and makes available the provided asset type, currency
// pair.
func (bot *Keep) ActivatePair(e exchange.IBotExchange, a asset.Item, p currency.Pair) error {
	base := e.GetBase()

	if err := base.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return err
	}

	// updated enabled pairs
	enabledpairs, err := base.CurrencyPairs.GetPairs(a, true)
	if err != nil {
		return err
	}

	enabledpairs = append(enabledpairs, p)
	base.CurrencyPairs.StorePairs(a, enabledpairs, true)

	// updated available pairs
	availablepairs, err := base.CurrencyPairs.GetPairs(a, false)
	if err != nil {
		return err
	}

	availablepairs = append(availablepairs, p)
	base.CurrencyPairs.StorePairs(a, availablepairs, false)

	return nil
}

// +----------------------------+
// | Copied from gocryptotrader |
// +----------------------------+

var (
	ErrNoExchangesLoaded    = errors.New("no exchanges have been loaded")
	ErrExchangeFailedToLoad = errors.New("exchange failed to load")
)

// loadExchange is an adapted version of GCT's Engine.LoadExchange.
// nolint: funlen, gocognit, gocyclo, cyclop
func (bot *Keep) loadExchange(exchCfg *config.Exchange, wg *sync.WaitGroup) error {
	exch, err := bot.ExchangeManager.NewExchangeByName(exchCfg.Name)
	if err != nil {
		return err
	}

	base := exch.GetBase()
	if base == nil {
		return ErrExchangeFailedToLoad
	}

	exch.SetDefaults()
	// overwrite whatever name the exchange wrapper has decided with the name that's in the config,
	// this is due to the exchange alias functionality that we offer over GCT.
	base.Name = exchCfg.Name

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

	// nolint: nestif
	if exchCfg.Features != nil {
		if bot.Settings.EnableExchangeWebsocketSupport &&
			base.Features.Supports.Websocket {
			exchCfg.Features.Enabled.Websocket = true

			if exchCfg.WebsocketTrafficTimeout <= 0 {
				What(log.Info().
					Str("exchange", exchCfg.Name).
					Dur("default", defaultWebsocketTrafficTimeout),
					"Websocket response traffic timeout value not set, setting default")

				exchCfg.WebsocketTrafficTimeout = defaultWebsocketTrafficTimeout
			}
		}

		if bot.Settings.EnableExchangeAutoPairUpdates &&
			base.Features.Supports.RESTCapabilities.AutoPairUpdates {
			exchCfg.Features.Enabled.AutoPairUpdates = true
		}

		if bot.Settings.DisableExchangeAutoPairUpdates {
			if base.Features.Supports.RESTCapabilities.AutoPairUpdates {
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

	if !bot.Settings.EnableExchangeHTTPRateLimiter {
		What(log.Info().
			Str("exchange", exch.GetName()),
			"Rate limiting has been turned off")

		if err := exch.DisableRateLimiter(); err != nil {
			What(log.Error().
				Err(err).
				Str("exchange", exch.GetName()),
				"unable to disable exchange rate limiter")
		}
	}

	exchCfg.Enabled = true

	if err := exch.Setup(exchCfg); err != nil {
		exchCfg.Enabled = false

		return err
	}

	bot.ExchangeManager.Add(exch)

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

		if err := exch.ValidateCredentials(context.TODO(), useAsset); err != nil {
			gctlog.Warnf(gctlog.ExchangeSys,
				"%s: Cannot validate credentials: %s\n",
				base.Name,
				err)
		}
	}

	if wg != nil {
		if err := exch.Start(wg); err != nil {
			return fmt.Errorf("unable to start exchange: %w", err)
		}
	} else {
		tempWG := sync.WaitGroup{}

		if err := exch.Start(&tempWG); err != nil {
			return fmt.Errorf("unable to start exchange: %w", err)
		}

		tempWG.Wait()
	}

	return nil
}

// setupExchanges is an (almost) unchanged copy of Engine.SetupExchanges.
//
//nolint
func (bot *Keep) setupExchanges() error {
	var wg sync.WaitGroup

	configs := bot.Config.GetAllExchangeConfigs()

	for x := range configs {
		if !configs[x].Enabled && !bot.Settings.EnableAllExchanges {
			What(log.Debug().
				Str("exchange", configs[x].Name),
				"exchange disabled")
			continue
		}
		wg.Add(1)
		go func(c config.Exchange) {
			defer wg.Done()
			err := bot.loadExchange(&c, &wg)
			if err != nil {
				What(log.Error().
					Err(err).
					Str("exchange", c.Name),
					"exchange load failed")

				return
			}

			What(log.Debug().
				Str("exchange", c.Name).
				Bool("authenticated", c.API.AuthenticatedSupport).
				Bool("verbose", c.Verbose),
				"exchange load failed")
		}(configs[x])
	}
	wg.Wait()
	if len(bot.GetExchanges()) == 0 {
		return ErrNoExchangesLoaded
	}
	return nil
}

// GetExchangeByName returns an exchange interface given it's name.
func (bot *Keep) GetExchangeByName(name string) (exchange.IBotExchange, error) {
	return bot.ExchangeManager.GetExchangeByName(name)
}

// GetEnabledPairAssetType returns the asset type that matches with the enabled provided currency pair,
// returns the first matching asset.
func (bot *Keep) GetEnabledPairAssetType(e exchange.IBotExchange, c currency.Pair) (asset.Item, error) {
	b := e.GetBase()

	assetTypes := b.GetAssetTypes(true)
	for i := range assetTypes {
		enabled, err := b.GetEnabledPairs(assetTypes[i])
		if err != nil {
			return asset.Spot, err
		}

		if enabled.Contains(c, true) {
			return assetTypes[i], nil
		}
	}

	return asset.Spot, ErrNoAssetType
}
