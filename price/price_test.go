package price

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrices(t *testing.T) {
	c := New(WithCoins("dero", "kaspa", "ethereum", "ergo", "monero"), WithVSCurrencies("usd", "eur", "chf"))
	err := c.LoadPrices()
	assert.NoError(t, err)
	client, ok := c.(*client)
	assert.True(t, ok)
	assert.NotNil(t, client.prices)

	p := c.GetPrices("ethereum")
	assert.NotNil(t, p)

	for _, price := range p {
		assert.NotNil(t, price)
		t.Log(price)
	}
}

func TestNoData(t *testing.T) {
	c := New(WithCoins("ergo"), WithVSCurrencies("usd"))
	err := c.LoadPrices()
	assert.NoError(t, err)
	client, ok := c.(*client)
	assert.True(t, ok)
	assert.NotNil(t, client.prices)

	p := c.GetPrices("no-data")
	assert.Nil(t, p)
}

func TestInvalidCoin(t *testing.T) {
	c := New(WithCoins("no-data"), WithVSCurrencies("usd"))
	err := c.LoadPrices()
	assert.NoError(t, err)
	client, ok := c.(*client)
	assert.True(t, ok)
	assert.NotNil(t, client.prices)

	p := c.GetPrices("no-data")
	assert.Nil(t, p)
}

func TestInvalidVSCurrency(t *testing.T) {
	c := New(WithCoins("dero"), WithVSCurrencies("no-data"))
	err := c.LoadPrices()
	assert.Error(t, err)

	p := c.GetPrices("dero")
	assert.Nil(t, p)
}

func TestClientLoop(t *testing.T) {
	c := New(WithCoins("ethereum"), WithVSCurrencies("chf"))
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		c.Start(time.Millisecond * 500)
	}()
	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 3)
		c.Close()
	}()
	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 1)
		p := c.GetPrices("ethereum")
		assert.NotNil(t, p)
		for _, price := range p {
			assert.NotNil(t, price)
			t.Log(price)
		}
	}()
	wg.Wait()
}

func TestClientWithCtx(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	c := New(WithCoins("ethereum"), WithVSCurrencies("chf"), WithContext(ctx))
	c.Start(time.Millisecond * 500)
}

func TestNoCoins(t *testing.T) {
	c := New(WithVSCurrencies("chf"))
	err := c.LoadPrices()
	assert.ErrorIs(t, err, ErrNoCoins)
}

func TestInvalidGet(t *testing.T) {
	c := New(WithCoins("ethereum"), WithVSCurrencies("chf"))
	err := c.LoadPrices()
	assert.NoError(t, err)
	p := c.GetPrices("no-data")
	assert.Nil(t, p)
}
