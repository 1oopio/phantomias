package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/stratumfarm/phantomias/config"
	"github.com/stratumfarm/phantomias/database"
)

func main() {
	// load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalln(fmt.Errorf("failed to load config: %w", err))
	}

	// create the database connection
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

}
