package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"
)

var MinerStatsMaxAge = time.Minute * 20

type MinerStatsSchema struct {
	ID              int64
	PoolID          string
	Miner           string
	Worker          string
	Hashrate        float64
	SharesPerSecond float64
	Created         time.Time
	HashrateType    string
}

type MinerWorkerPerformanceStats struct {
	PoolID           string
	Miner            string
	Worker           string
	Hashrate         *float64
	HashrateType     string
	SharesPerSecond  *float64
	Created          time.Time
	ReportedHashrate *float64
}

type MinerWorkerPerformanceStatsEntity struct {
	MinerWorkerPerformanceStats
	Partition int
}

type MinerPerformanceStats struct {
	Miner           string
	Hashrate        float64
	SharesPerSecond float64
}

type WorkerPerformanceStats struct {
	Hashrate         *float64
	ReportedHashrate *float64
	SharesPerSecond  *float64
}

type WorkerPerformanceStatsContainer struct {
	Created time.Time
	Workers map[string]*WorkerPerformanceStats
}

type MinerStats struct {
	PendingShares  *float64
	PendingBalance *float64
	TotalPaid      *float64
	TodayPaid      *float64
	LastPayment    *Payment
	Performance    *WorkerPerformanceStatsContainer
}

func (d *DB) PagePoolMinersByHashrate(ctx context.Context, poolID string, from time.Time, page int, pageSite int) ([]MinerPerformanceStats, error) {
	var miners []MinerPerformanceStats
	err := d.sql.SelectContext(ctx, &miners, `
		WITH tmp AS
		(
			SELECT
				ms.miner,
				ms.hashrate,
				ms.sharespersecond,
				ROW_NUMBER() OVER(PARTITION BY ms.miner ORDER BY ms.hashrate DESC) AS rk
			FROM (SELECT miner, SUM(hashrate) AS hashrate, SUM(sharespersecond) AS sharespersecond
				FROM minerstats
				WHERE poolid = $1 AND created >= $2 AND hashratetype = 'actual' GROUP BY miner, created) ms
		)
		SELECT t.miner, t.hashrate, t.sharespersecond
		FROM tmp t
		WHERE t.rk = 1
		ORDER by t.hashrate DESC
		OFFSET $3 FETCH NEXT $4 ROWS ONLY;
	`, poolID, from, page*pageSite, pageSite)
	return miners, err
}

func (d *DB) GetMinersCount(ctx context.Context, poolID string, from time.Time) (uint, error) {
	var count uint
	err := d.sql.GetContext(ctx, &count, `
		WITH tmp AS
		(
			SELECT
				ms.miner,
				ms.hashrate,
				ms.sharespersecond,
				ROW_NUMBER() OVER(PARTITION BY ms.miner ORDER BY ms.hashrate DESC) AS rk
			FROM (SELECT miner, SUM(hashrate) AS hashrate, SUM(sharespersecond) AS sharespersecond
				FROM minerstats
				WHERE poolid = $1 AND created >= $2 AND hashratetype = 'actual' GROUP BY miner, created) ms
		)
		SELECT count(t.miner)
		FROM tmp t
		WHERE t.rk = 1;
	`, poolID, from)
	return count, err
}

func (d *DB) GetMinerStats(ctx context.Context, poolID string, miner string) (*MinerStats, error) {
	var stats MinerStats
	err := d.sql.GetContext(ctx, &stats, `
	SELECT 
		(SELECT SUM(difficulty) FROM shares WHERE poolid = $1 AND miner = $2) AS pendingshares,
		(SELECT amount FROM balances WHERE poolid = $1 AND address = $2) AS pendingbalance,
		(SELECT SUM(amount) FROM payments WHERE poolid = $1 and address = $2) as totalpaid,
		(SELECT SUM(amount) FROM payments WHERE poolid = $1 and address = $2 and created >= date_trunc('day', now())) as todaypaid;
	`, poolID, miner)
	if err != nil {
		return nil, fmt.Errorf("failed to get miner stats: %w", err)
	}

	stats.LastPayment = new(Payment)
	err = d.sql.GetContext(ctx, stats.LastPayment, `
	SELECT * FROM payments WHERE poolid = $1 AND address = $2 ORDER BY created DESC LIMIT 1;
	`, poolID, miner)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("[db][error] no last payment found for miner", miner)
			stats.LastPayment = nil
		} else {
			return nil, fmt.Errorf("failed to get miner last payment: %w", err)
		}
	}

	var lastUpdated time.Time
	err = d.sql.GetContext(ctx, &lastUpdated, `
	SELECT created FROM minerstats WHERE poolid = $1 AND miner = $2 AND hashratetype = 'actual' ORDER BY created DESC LIMIT 1;
	`, poolID, miner)
	if err != nil {
		return nil, fmt.Errorf("failed to get miner last updated: %w", err)
	}

	if time.Since(lastUpdated) > MinerStatsMaxAge {
		return &stats, nil
	}

	lastReportedUpdate := time.Now().Add(-MinerStatsMaxAge)

	var performanceStats []*MinerWorkerPerformanceStats
	err = d.sql.SelectContext(ctx, &performanceStats, `
	SELECT ms.created, ms.poolid, ms.miner, ms.worker,
		(SELECT hashrate FROM minerstats hms WHERE hms.hashrateType = 'actual' AND hms.worker = ms.worker AND hms.miner = ms.miner AND hms.poolid = ms.poolid AND hms.created = ms.created ORDER BY hms.created DESC LIMIT 1),
		(SELECT sharespersecond FROM minerstats hms WHERE hms.hashrateType = 'actual' AND hms.worker = ms.worker AND hms.miner = ms.miner AND hms.poolid = ms.poolid AND hms.created = ms.created ORDER BY hms.created DESC LIMIT 1),
		(SELECT hashrate FROM minerstats hms WHERE hms.hashrateType = 'reported' AND hms.worker = ms.worker AND hms.miner = ms.miner AND hms.poolid = ms.poolid AND hms.created > $1 ORDER BY hms.created DESC LIMIT 1) as reportedHashrate
	FROM minerstats ms WHERE ms.poolid = $2 AND ms.miner = $3 AND ms.created = $4 GROUP BY ms.poolid, ms.miner, ms.worker, ms.created;
	`, lastReportedUpdate, poolID, miner, lastUpdated)
	if err != nil {
		return nil, fmt.Errorf("failed to get miner performance stats: %w", err)
	}
	if len(performanceStats) <= 0 {
		return &stats, nil
	}
	stats.Performance = minerWorkerPerformanceStatsToWorkerPerformanceStatsContainer(performanceStats)

	return &stats, err
}

func minerWorkerPerformanceStatsToWorkerPerformanceStatsContainer(stats []*MinerWorkerPerformanceStats) *WorkerPerformanceStatsContainer {
	container := &WorkerPerformanceStatsContainer{
		Created: stats[0].Created,
		Workers: make(map[string]*WorkerPerformanceStats),
	}

	for _, stat := range stats {
		container.Workers[stat.Worker] = &WorkerPerformanceStats{
			Hashrate:         stat.Hashrate,
			SharesPerSecond:  stat.SharesPerSecond,
			ReportedHashrate: stat.ReportedHashrate,
		}
	}

	return container
}

func (d *DB) GetMinerPerformanceBetweenTenMinutely(ctx context.Context, poolID, miner string, start, end time.Time) ([]*WorkerPerformanceStatsContainer, error) {
	var stats []*MinerWorkerPerformanceStatsEntity
	err := d.sql.SelectContext(ctx, &stats, `
	SELECT date_trunc('hour', x.created) AS created,
           (extract(minute FROM x.created)::int / 10) AS partition,
           x.worker, AVG(x.hs) AS hashrate, AVG(x.rhs) AS reportedhashrate, AVG(x.sharespersecond) AS sharespersecond
           FROM (
           SELECT created, hashrate as hs, null as rhs, sharespersecond, worker FROM minerstats WHERE poolid = $1 AND miner = $2 AND created >= $3 AND created <= $4 AND hashratetype = 'actual'
           UNION 
           SELECT created, null as hs, hashrate as rhs, null as sharespersecond, worker FROM minerstats WHERE poolid = $1 AND miner = $2 AND created >= $3 AND created <= $4 AND hashratetype = 'reported'
           ) as x
           GROUP BY 1, 2, worker
           ORDER BY 1, 2, worker
		   `, poolID, miner, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get miner performance stats: %w", err)
	}
	for _, stat := range stats {
		stat.Created = stat.Created.Add(time.Duration(stat.Partition) * 10 * time.Minute)
	}
	return entitiesByDate(stats), nil
}

func entitiesByDate(entities []*MinerWorkerPerformanceStatsEntity) []*WorkerPerformanceStatsContainer {
	var stats []*WorkerPerformanceStatsContainer
	statsByDate := make(map[time.Time]*WorkerPerformanceStatsContainer)
	for _, entity := range entities {
		if _, ok := statsByDate[entity.Created]; !ok {
			statsByDate[entity.Created] = &WorkerPerformanceStatsContainer{
				Created: entity.Created,
				Workers: make(map[string]*WorkerPerformanceStats),
			}
		}
		statsByDate[entity.Created].Workers[entity.Worker] = &WorkerPerformanceStats{
			Hashrate:         entity.Hashrate,
			ReportedHashrate: entity.ReportedHashrate,
			SharesPerSecond:  entity.SharesPerSecond,
		}
	}
	for _, stat := range statsByDate {
		stats = append(stats, stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Created.Before(stats[j].Created)
	})
	return stats
}
