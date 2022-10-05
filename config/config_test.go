package config_test

import (
	"testing"
	"time"

	_ "github.com/stratumfarm/phantomias/cmd"
	"github.com/stratumfarm/phantomias/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := config.Load("testdata/config.yml")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Proxy)
	assert.NotNil(t, cfg.Miningcore)
	assert.NotNil(t, cfg.Price)
	assert.NotNil(t, cfg.Metrics)

	assert.Equal(t, "0.0.0.0:3000", cfg.Proxy.Listen)
	assert.Equal(t, time.Duration(time.Minute), cfg.Proxy.CacheTTL)

	assert.Equal(t, "http://localhost:5000", cfg.Miningcore.URL)
	assert.Equal(t, "ws://localhost:5000/notifications", cfg.Miningcore.WS)

	assert.Equal(t, []string{"ethereum", "ergo"}, cfg.Price.Coins)
	assert.Equal(t, []string{"usd", "eur", "chf"}, cfg.Price.VSCurrencies)

	assert.Equal(t, "0.0.0.0:3001", cfg.Metrics.Listen)
	assert.Equal(t, "/metrics", cfg.Metrics.Endpoint)
	assert.Equal(t, true, cfg.Metrics.Enabled)
	assert.Equal(t, "metrics", cfg.Metrics.User)
	assert.Equal(t, "metricspasswd", cfg.Metrics.Password)

	assert.Equal(t, "postgreshost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "postgresuser", cfg.DB.User)
	assert.Equal(t, "postgrespassword", cfg.DB.Password)
	assert.Equal(t, "postgresdb", cfg.DB.Dbname)

	assert.Len(t, cfg.Pools, 1)
	assert.Equal(t, "dero1", cfg.Pools[0].ID)
	assert.Equal(t, "dero", cfg.Pools[0].Type)
	assert.Equal(t, "ws://deronode:10102/ws", cfg.Pools[0].RPC)
	assert.Equal(t, "AstroBWT/v3", cfg.Pools[0].Algorithm)
	assert.Equal(t, float64(4), cfg.Pools[0].Fee)
	assert.Equal(t, "PPLNS", cfg.Pools[0].FeeType)
	assert.Equal(t, "Dero", cfg.Pools[0].Name)
	assert.Equal(t, "DERO", cfg.Pools[0].Coin)
	assert.Equal(t, "https://explorer.dero.io/block/%d", cfg.Pools[0].BlockLink)
	assert.Equal(t, "https://explorer.dero.io/tx/%s", cfg.Pools[0].TxLink)
	assert.Equal(t, "", cfg.Pools[0].AddressLink)

	assert.Len(t, cfg.Pools[0].Ports, 1)
	assert.Equal(t, float64(42000), cfg.Pools[0].Ports["4300"].Difficulty)
	assert.Equal(t, true, cfg.Pools[0].Ports["4300"].VarDiff)
	assert.Equal(t, true, cfg.Pools[0].Ports["4300"].TLS)
	assert.Equal(t, true, cfg.Pools[0].Ports["4300"].TLSAuto)

	assert.Equal(t, "deroxyz", cfg.Pools[0].Address)
	assert.Equal(t, 0.2, cfg.Pools[0].MinPayout)
	assert.Equal(t, float64(10), cfg.Pools[0].ShareMultiplier)

}

func TestLoadNoFileConfig(t *testing.T) {
	cfg, err := config.Load("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Proxy)
	assert.NotNil(t, cfg.Miningcore)

}

func TestLoadInvalidFile(t *testing.T) {
	_, err := config.Load("testdata/invalid.txt")
	assert.Error(t, err)
}
