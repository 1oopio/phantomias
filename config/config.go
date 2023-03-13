package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const defaultConfigFile = "config.yml"

// Config represents the config
type Config struct {
	Pools      []*Pool     `mapstructure:"pools"`
	DB         *DB         `mapstructure:"db"`
	API        *API        `mapstructure:"api"`
	Miningcore *Miningcore `mapstructure:"miningcore"`
	Price      *Price      `mapstructure:"price"`
	Metrics    *Metrics    `mapstructure:"metrics"`
}

// DB represents the database config
type DB struct {
	Host     string `mapstructure:"host"`     // database host
	Port     int    `mapstructure:"port"`     // database port
	User     string `mapstructure:"user"`     // database user
	Password string `mapstructure:"password"` // database password
	Dbname   string `mapstructure:"dbname"`   // database name
	SSLMode  string `mapstructure:"ssl"`      // ssl mode
}

// Pool represents the config for a single pool
type Pool struct {
	ID              string          `mapstructure:"id"`               // pool id
	Enabled         bool            `mapstructure:"enabled"`          // pool enabled
	Type            string          `mapstructure:"type"`             // coinfamily
	RPC             string          `mapstructure:"rpc"`              // rpc url
	Algorithm       string          `mapstructure:"algorithm"`        // algorithm
	Name            string          `mapstructure:"name"`             // pool name
	Coin            string          `mapstructure:"coin"`             // coin name
	Fee             float64         `mapstructure:"fee"`              // pool fee
	FeeType         string          `mapstructure:"fee_type"`         // pool fee type
	BlockLink       string          `mapstructure:"block_link"`       // block link (explorer)
	TxLink          string          `mapstructure:"tx_link"`          // transaction link (explorer)
	AddressLink     string          `mapstructure:"address_link"`     // address link (explorer)
	Ports           map[string]Port `mapstructure:"ports"`            // ports
	Address         string          `mapstructure:"address"`          // address of the pool
	MinPayout       float64         `mapstructure:"min_payout"`       // minimum payout
	ShareMultiplier float64         `mapstructure:"share_multiplier"` // share multiplier
}

// Port represents a pool port
type Port struct {
	Difficulty float64 `mapstructure:"difficulty"` // difficulty
	VarDiff    bool    `mapstructure:"var_diff"`   // var diff
	TLS        bool    `mapstructure:"tls"`        // tls
	TLSAuto    bool    `mapstructure:"tls_auto"`   // tls auto
}

// API represents the configuration for the proxy.
type API struct {
	Listen            string        `mapstructure:"listen"`              // listening address e.g. 127.0.0.1:8080
	CacheTTL          time.Duration `mapstructure:"cache_ttl"`           // cache TTL
	CertFile          string        `mapstructure:"cert_file"`           // path to the tls certificate
	CertKey           string        `mapstructure:"cert_key"`            // path to the tls key
	TrustedProxyCheck bool          `mapstructure:"trusted_proxy_check"` // allow requests only from trusted proxies
	TrustedProxies    []string      `mapstructure:"trusted_proxies"`     // a list of trusted proxy IPs
}

// MiningcoreConfig represents the configuration for the miningcore client.
type Miningcore struct {
	URL       string        `mapstructure:"url"`        // url of the miningcore api server
	WS        string        `mapstructure:"ws"`         // url of the miningcore websocket server
	IgnoreTLS bool          `mapstructure:"ignore_tls"` // ignore invalid tls certificates
	Timeout   time.Duration `mapstructure:"timeout"`    // timeout for the api client
}

// PriceConfig represents the configuration for the price service.
type Price struct {
	Coins        []string `mapstructure:"coins"`
	VSCurrencies []string `mapstructure:"vs_currencies"`
}

// MetricsConfig represents the configuration for the metrics service.
type Metrics struct {
	Enabled  bool   `mapstructure:"enabled"`  // enable metrics service
	Listen   string `mapstructure:"listen"`   // listening address
	Endpoint string `mapstructure:"endpoint"` // endpoint for the metrics server
	User     string `mapstructure:"user"`     // user for metrics
	Password string `mapstructure:"password"` // password for metrics
}

// Load loads the config file.
// It searches in the following locations:
//
// /etc/phantomias/config.yml,
// $HOME/.config/phantomias/config.yml,
// config.yml
//
// command arguments will overwrite the value from the config
func Load(path string) (cfg *Config, err error) {
	if path != "" {
		return load(path)
	}
	for _, f := range [4]string{
		".config.yml",
		"config.yml",
	} {
		cfg, err = load(f)
		if err != nil && os.IsNotExist(err) {
			err = nil
			continue
		} else if err != nil && errors.As(err, &viper.ConfigFileNotFoundError{}) {
			err = nil
			continue
		}
	}
	if cfg == nil {
		return cfg, viper.Unmarshal(&cfg)
	}
	return
}

func load(file string) (cfg *Config, err error) {
	viper.SetConfigName(file)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/phantomias")
	viper.AddConfigPath("/etc/phantomias/")

	viper.SetEnvPrefix("phantomias")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.MustBindEnv("db.password")

	if err = viper.ReadInConfig(); err != nil {
		return
	}
	if err = viper.Unmarshal(&cfg); err != nil {
		return
	}
	return
}
