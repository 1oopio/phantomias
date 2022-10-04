package database

import (
	"context"
	"fmt"
	"time"
)

func (d *DB) GetWorkerPerformanceBetweenTenMinutely(ctx context.Context, poolID, miner, worker string, start, end time.Time) ([]*PerformanceStats, error) {
	var stats []*MinerWorkerPerformanceStatsEntity
	err := d.sql.SelectContext(ctx, &stats, `
	SELECT date_trunc('hour', x.created) AS created,
		(extract(minute FROM x.created)::int / 10) AS partition,
		AVG(x.hs) AS hashrate, AVG(x.rhs) AS reportedhashrate, AVG(x.sharespersecond) AS sharespersecond
		FROM (
			SELECT created, hashrate as hs, null as rhs, sharespersecond, worker FROM minerstats WHERE poolid = $1 AND miner = $2 AND worker = $3 AND created >= $4 AND created <= $5 AND hashratetype = 'actual'
			UNION 
			SELECT created, null as hs, hashrate as rhs, null as sharespersecond, worker FROM minerstats WHERE poolid = $1 AND miner = $2 AND worker = $3 AND created >= $4 AND created <= $5 AND hashratetype = 'reported'
		) as x
		GROUP BY 1, 2
		ORDER BY 1, 2;
		`, poolID, miner, worker, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker performance stats: %w", err)
	}
	for _, stat := range stats {
		stat.Created = stat.Created.Add(time.Duration(stat.Partition) * 10 * time.Minute)
	}
	return entitiesByDate(stats), nil
}
