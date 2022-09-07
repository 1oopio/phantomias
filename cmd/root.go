package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stratumfarm/phantomias/config"
	"github.com/stratumfarm/phantomias/database"
	"github.com/stratumfarm/phantomias/metrics"
	"github.com/stratumfarm/phantomias/price"
	"github.com/stratumfarm/phantomias/version"
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
	rootCmd.Flags().Duration("cache-ttl", time.Second*5, "TTL for the api cache")
	rootCmd.Flags().String("cert-file", "", "path to the tls certificate")
	rootCmd.Flags().String("cert-key", "", "path to the tls key")
	rootCmd.Flags().Bool("trusted-proxy-check", false, "allow requests only from trusted proxies")
	rootCmd.Flags().StringArray("trusted-proxies", nil, "a list of trusted proxy IPs")
	rootCmd.Flags().Bool("enable-metrics", false, "enable the metrics dashboard")

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

	viper.BindPFlag("proxy.listen", rootCmd.Flags().Lookup("listen"))
	viper.BindPFlag("proxy.cache_ttl", rootCmd.Flags().Lookup("cache-ttl"))
	viper.BindPFlag("proxy.cert_file", rootCmd.Flags().Lookup("cert-file"))
	viper.BindPFlag("proxy.cert_key", rootCmd.Flags().Lookup("cert-key"))
	viper.BindPFlag("proxy.trusted_proxy_check", rootCmd.Flags().Lookup("trusted-proxy-check"))
	viper.BindPFlag("proxy.trusted_proxies", rootCmd.Flags().Lookup("trusted-proxies"))
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

	// connect to the database
	db := database.New(cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Dbname)
	if err := db.Connect(); err != nil {
		log.Fatalln(fmt.Errorf("failed to connect to database: %w", err))
	}
	defer db.Close()

	log.Println("Connected to database")

	/* poolStats, err := db.GetLastPoolStats(context.Background(), "dero1")
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get pool stats: %w", err))
	}
	log.Println(poolStats)

	poolBlockTime, err := db.GetLastPoolBlockTime(context.Background(), "dero1")
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get pool block time: %w", err))
	}
	log.Println(poolBlockTime)

	blockCount, err := db.GetPoolBlockCount(context.Background(), "dero1")
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get pool block count: %w", err))
	}
	log.Println(blockCount)

	poolPayments, err := db.GetTotalPoolPayments(context.Background(), "dero1")
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get pool payments: %w", err))
	}
	log.Println(poolPayments)

	minersByHashrate, err := db.PagePoolMinersByHashrate(context.Background(), "dero1", time.Now().Add(-1*time.Hour), 0, 15)
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get miners by hashrate: %w", err))
	}
	log.Println(minersByHashrate)

	poolPerformanceBetween, err := db.GetPoolPerformanceBetween(context.Background(), "dero1", database.IntervalHour, time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get pool performance between: %w", err))
	}
	log.Println(poolPerformanceBetween)

	blocks, err := db.PageBlocks(context.Background(), "dero1", []database.BlockStatus{database.BlockStatusConfirmed}, 0, 15)
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get blocks: %w", err))
	}
	log.Println(blocks)

	payments, err := db.PagePayments(context.Background(), "dero1", "", 0, 15)
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get payments: %w", err))
	}
	log.Println(payments)

	paymentsCount, err := db.GetPaymentsCount(context.Background(), "dero1", "")
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to get payments count: %w", err))
	}
	log.Println(paymentsCount)

	stats, err := db.GetMinerStats(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf")
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(stats) */

	/* log.Println(db.PageBalanceChanges(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf", 0, 15))
	log.Println(db.GetBalanceChangesCount(context.Background(), "dero1", ""))
	log.Println(db.GetBalanceChangesCount(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf"))
	log.Println(db.GetMinerPaymentsByDayCount(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf"))
	log.Println(db.PageMinerPaymentsByDay(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf", 1, 15))
	log.Println(db.GetSettings(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf")) */
	ips, err := db.GetRecentyUsedIpAddresses(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf")
	if err != nil {
		log.Fatalln(err)
	}
	for _, ip := range ips {
		fmt.Println(net.ParseIP(ip))
	}

	minerPerf, err := db.GetMinerPerformanceBetweenTenMinutely(context.Background(), "dero1", "dero1qyg454q3fqj607x8yayfkvznftf4kfk3hdu2206n289znj5d4rqfcqg8v4gqf", time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(minerPerf)
	for _, perf := range minerPerf {
		log.Println(perf.Created)
		for k, v := range perf.Workers {
			log.Println(k, v)
		}
	}

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
}
