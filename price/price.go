package price

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/esenmx/gocko"
)

const defaultFetchInterval = time.Minute

var (
	// ErrNoCoins is returned when no coins are specified to load prices
	ErrNoCoins = errors.New("no coins specified")
)

// Opts is a function that can be passed to New to configure the client
type Opts func(c *client)

// WithCoins sets the coins to load prices for
func WithCoins(coins ...string) Opts {
	return func(c *client) {
		sanitizedCoins := make([]string, 0, len(coins))
		for _, coin := range coins {
			sanitizedCoins = append(sanitizedCoins, strings.ToLower(strings.ReplaceAll(coin, " ", "-")))
		}
		c.coins = sanitizedCoins
	}
}

// WithVSCurrencies sets the currencies in which to load prices
func WithVSCurrencies(vsCurrencies ...string) Opts {
	return func(c *client) {
		c.vsCurrencies = vsCurrencies
	}
}

// WithContext sets the context to use for the client
func WithContext(ctx context.Context) Opts {
	return func(c *client) {
		c.parentCtx = ctx
	}
}

// Price contains the price of a coin in a specified currency
type Price struct {
	VSCurrency               string
	Coin                     string
	Price                    float64
	PriceChangePercentage24H float64
}

// Client is a client for fetching prices
type Client interface {
	// Start starts fetching prices at the given interval
	Start(interval ...time.Duration)
	// LoadPrices fetches the prices at rest and caches them
	// it returns an error if it fails to fetch the prices
	LoadPrices() error
	// GetPrices returns the prices for the given coin
	GetPrices(coin string) []*Price
	// Close closes the client and stops fetching prices
	Close()
}

type client struct {
	parentCtx    context.Context
	ctx          context.Context
	cancel       context.CancelFunc
	vsCurrencies []string
	coins        []string

	gocko  *gocko.Client
	prices map[string][]*Price
	mu     sync.Mutex
}

// New creates a new client for fetching prices
func New(opts ...Opts) Client {
	c := &client{
		parentCtx:    context.Background(),
		vsCurrencies: make([]string, 0),
		coins:        make([]string, 0),
		gocko:        gocko.NewClient(),
		prices:       make(map[string][]*Price),
	}
	for _, opt := range opts {
		opt(c)
	}
	c.ctx, c.cancel = context.WithCancel(c.parentCtx)
	return c
}

// LoadPrices fetches the prices and caches them
func (c *client) LoadPrices() error {
	prices := make([]*Price, 0, len(c.vsCurrencies))
	for _, currency := range c.vsCurrencies {
		p, err := c.loadPrices(currency)
		if err != nil {
			return err
		}
		prices = append(prices, p...)
	}
	c.mu.Lock()
	c.prices = mapCoins(prices)
	c.mu.Unlock()
	return nil
}

func mapCoins(prices []*Price) map[string][]*Price {
	c := make(map[string][]*Price)
	for _, price := range prices {
		c[price.Coin] = append(c[price.Coin], price)
	}
	return c
}

func (c *client) loadPrices(VsCurrency string) ([]*Price, error) {
	if len(c.coins) == 0 {
		return nil, ErrNoCoins
	}
	markets, err := c.gocko.CoinsMarkets(gocko.CoinsMarketsParams{
		VsCurrency: VsCurrency,
		Ids:        c.coins,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get markets: %w", err)
	}
	prices := make([]*Price, 0, len(markets))
	for _, market := range markets {
		prices = append(prices, &Price{
			VSCurrency:               VsCurrency,
			Price:                    market.CurrentPrice,
			Coin:                     market.Id,
			PriceChangePercentage24H: market.PriceChangePercentage24H,
		})
	}
	return prices, nil
}

// GetPrices returns the prices for the given coin
func (c *client) GetPrices(coin string) []*Price {
	coin = strings.ToLower(strings.ReplaceAll(coin, " ", "-"))
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.prices[coin]) == 0 {
		return nil
	}
	return c.prices[coin]
}

// Start starts fetching prices at the given interval
func (c *client) Start(interval ...time.Duration) {
	i := defaultFetchInterval
	if len(interval) > 0 {
		i = interval[0]
	}
	if err := c.LoadPrices(); err != nil {
		log.Printf("failed to load initial prices: %s", err)
	}
	log.Printf("[price][client] starting with interval %s", i)

	ticker := time.NewTicker(i)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := c.LoadPrices(); err != nil {
				log.Printf("failed to load prices: %s", err)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// Close closes the client and stops fetching prices
func (c *client) Close() {
	c.cancel()
}
