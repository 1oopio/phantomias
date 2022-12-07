package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type WorkerStats struct {
	Hashrate        *float64
	SharesPerSecond *float64
}

func (d *DB) GetWorkerPerformanceBetweenTenMinutely(ctx context.Context, poolID, miner, worker string, start, end time.Time) ([]*PerformanceStatsEntity, error) {
	var stats []*PerformanceStatsEntity
	err := d.sql.SelectContext(ctx, &stats, `
	SELECT * FROM
	(
	SELECT 
		date_trunc('hour', x.created) AS created,
		(extract(minute FROM x.created)::int / 10) AS partition,
		AVG(x.hs) AS hashrate, 
		AVG(x.rhs) AS reportedhashrate, 
		AVG(x.sharespersecond) AS sharespersecond
	FROM (
		SELECT created, hashrate as hs, null as rhs, sharespersecond, worker 
		FROM minerstats 
		WHERE 
			poolid = $1 AND 
			miner = $2 AND 
			worker = $3 AND
			created >= $4 AND 
			created <= $5
	UNION 
		SELECT created, null as hs, hashrate as rhs, null as sharespersecond, worker 
		FROM reported_hashrate 
		WHERE 
			poolid = $1 AND 
			miner = $2 AND
			worker = $3 AND 
			created >= $4 AND 
			created <= $5
	) as x
	GROUP BY 1, 2
	ORDER BY 1, 2
	) as res
	WHERE 
		res.hashrate IS NOT NULL OR 
		res.reportedhashrate IS NULL;
	`, poolID, miner, worker, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker performance stats: %w", err)
	}
	for _, stat := range stats {
		stat.Created = stat.Created.Add(time.Duration(stat.Partition) * 10 * time.Minute)
	}
	return stats, nil
}

func (d *DB) GetWorkerStats(ctx context.Context, poolID, miner, worker string) (WorkerStats, error) {
	var stats WorkerStats
	err := d.sql.GetContext(ctx, &stats, `
	SELECT
		hashrate,
		sharespersecond
	FROM minerstats
	WHERE
		poolid = $1 AND
		miner = $2 AND
		worker = $3
	ORDER BY
		created DESC
	LIMIT 1;
	`, poolID, miner, worker)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return WorkerStats{}, fmt.Errorf("failed to get worker stats: %w", err)
	}
	return stats, nil
}
