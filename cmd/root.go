package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/1oopio/phantomias/api"
	"github.com/1oopio/phantomias/config"
	"github.com/1oopio/phantomias/database"
	"github.com/1oopio/phantomias/metrics"
	"github.com/1oopio/phantomias/price"
	"github.com/1oopio/phantomias/version"
	"github.com/1oopio/phantomias/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stratumfarm/go-miningcore-client"
)

var rootCmdFlags struct {
	config string
}

var rootCmd = &cobra.Command{
	Use:     "phantomias",
	Short:   "start the miningcore api proxy",
	Version: version.Version,
	Run:     root,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.Flags().StringVarP(&rootCmdFlags.config, "config", "c", "", "path to the config file")

	rootCmd.Flags().String("listen", "0.0.0.0:8080", "listening address")
	rootCmd.Flags().Duration("cache-ttl", time.Second*90, "TTL for the api cache")
	rootCmd.Flags().String("cert-file", "", "path to the tls certificate")
	rootCmd.Flags().String("cert-key", "", "path to the tls key")
	rootCmd.Flags().Bool("trusted-proxy-check", false, "allow requests only from trusted proxies")
	rootCmd.Flags().StringArray("trusted-proxies", nil, "a list of trusted proxy IPs")
	rootCmd.Flags().Bool("enable-metrics", false, "enable the metrics dashboard")

	rootCmd.Flags().String("database-sslmode", "require", "database sslmode (pgsql)")

	rootCmd.Flags().String("miningcore-url", "", "url of the miningcore api")
	rootCmd.Flags().Bool("miningcore-ignore-tls", false, "ignore invalid tls configuration")
	rootCmd.Flags().String("miningcore-ws", "", "url of the miningcore websocket api")
	rootCmd.Flags().Duration("miningcore-timeout", time.Second*5, "timeout for the miningcore api")

	rootCmd.Flags().StringArray("price-coins", nil, "a list of coins to load prices for")
	rootCmd.Flags().StringArray("price-vscurrencies", nil, "a list of currencies in which to load prices")

	rootCmd.Flags().Bool("metrics-enabled", false, "enable prometheus metrics")
	rootCmd.Flags().String("metrics-listen", "0.0.0.0:8081", "listening address for the metrics server")
	rootCmd.Flags().String("metrics-endpoint", "/metrics", "the endpoint to fetch metrics from")
	rootCmd.Flags().String("metrics-user", "", "user for metrics")
	rootCmd.Flags().String("metrics-password", "", "password for metrics")

	viper.BindPFlag("api.listen", rootCmd.Flags().Lookup("listen"))
	viper.BindPFlag("api.cache_ttl", rootCmd.Flags().Lookup("cache-ttl"))
	viper.BindPFlag("api.cert_file", rootCmd.Flags().Lookup("cert-file"))
	viper.BindPFlag("api.cert_key", rootCmd.Flags().Lookup("cert-key"))
	viper.BindPFlag("api.trusted_proxy_check", rootCmd.Flags().Lookup("trusted-proxy-check"))
	viper.BindPFlag("api.trusted_proxies", rootCmd.Flags().Lookup("trusted-proxies"))
	viper.BindPFlag("database.sslmode", rootCmd.Flags().Lookup("database-sslmode"))
	viper.BindPFlag("miningcore.url", rootCmd.Flags().Lookup("miningcore-url"))
	viper.BindPFlag("miningcore.ignore_tls", rootCmd.Flags().Lookup("miningcore-ignore-tls"))
	viper.BindPFlag("miningcore.ws", rootCmd.Flags().Lookup("miningcore-ws"))
	viper.BindPFlag("miningcore.timeout", rootCmd.Flags().Lookup("miningcore-timeout"))
	viper.BindPFlag("price.coins", rootCmd.Flags().Lookup("price-coins"))
	viper.BindPFlag("price.vs_currencies", rootCmd.Flags().Lookup("price-vscurrencies"))
	viper.BindPFlag("metrics.listen", rootCmd.Flags().Lookup("metrics-listen"))
	viper.BindPFlag("metrics.endpoint", rootCmd.Flags().Lookup("metrics-endpoint"))
	viper.BindPFlag("metrics.enabled", rootCmd.Flags().Lookup("metrics-enabled"))
	viper.BindPFlag("metrics.user", rootCmd.Flags().Lookup("metrics-user"))
	viper.BindPFlag("metrics.password", rootCmd.Flags().Lookup("metrics-password"))
}

func Execute() error {
	return rootCmd.Execute()
}

func root(cmd *cobra.Command, args []string) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// load the config
	var err error
	cfg, err := config.Load(rootCmdFlags.config)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(cfg.DB.Password)

	// connect to the database
	db := database.New(cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Dbname, cfg.DB.SSLMode)
	if err := db.Connect(); err != nil {
		log.Fatalln(fmt.Errorf("failed to connect to database: %w", err))
	}
	defer db.Close()
	log.Println("Connected to database")

	// metrics
	var metricsMiddleware fiber.Handler
	if cfg.Metrics.Enabled {
		metricsServer := metrics.New(cfg.Metrics, metrics.WithContext(cmd.Context()))
		defer metricsServer.Close()

		go func() {
			if err := metricsServer.Start(); err != nil {
				log.Fatalln(err)
			}
		}()
		metricsMiddleware = metricsServer.Fiber()
	}
	_ = metricsMiddleware

	// fetch price data from coingecko
	priceClient := price.New(
		price.WithContext(cmd.Context()),
		price.WithCoins(cfg.Price.Coins...),
		price.WithVSCurrencies(cfg.Price.VSCurrencies...),
	)
	defer priceClient.Close()
	go func() {
		priceClient.Start()
	}()

	// create the miningcore client
	mcOpts := []miningcore.ClientOpts{
		miningcore.WithJSONEncoder(json.Marshal),
		miningcore.WithJSONDecoder(json.Unmarshal),
		miningcore.WithTimeout(cfg.Miningcore.Timeout),
	}
	if cfg.Miningcore.IgnoreTLS {
		mcOpts = append(mcOpts, miningcore.WithoutTLSVerfiy())
	}
	mc := miningcore.New(
		cfg.Miningcore.URL,
		mcOpts...,
	)

	// start the api server
	api := api.New(context.Background(), cfg.API, cfg.Pools, mc, db, priceClient, metricsMiddleware)
	defer api.Close()

	go func() {
		if err := api.Start(); err != nil {
			log.Fatalln(err)
		}
	}()

	// start the websocket relay
	wsc := ws.New(cfg.Miningcore.WS, api.BroadcastChan())
	defer wsc.Close()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				if err := wsc.Listen(done); err != nil {
					//log.Fatalln(err)
					log.Println("[err] failed to start the websocket relay, will try again in 30 seconds...")
				}
				time.Sleep(time.Second * 30)
			}
		}
	}()

	<-done
	log.Println("shutting down...")
}
