package database

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type PoolStats struct {
	ID                   int64
	PoolID               string
	ConnectedMiners      int32
	PoolHashrate         float64
	SharesPerSecond      float64
	NetworkHashrate      float64
	NetworkDifficulty    float64
	LastNetworkBlockTime time.Time
	BlockHeight          int64
	ConnectedPeers       int32
	Created              time.Time
}

type AggregatedPoolStats struct {
	PoolHashrate         float64
	ConnectedMiners      int
	ValidSharesPerSecond int
	NetworkHashrate      float64
	NetworkDifficulty    float64
	Created              time.Time
}

func (d *DB) GetLastPoolStats(ctx context.Context, poolID string) (*PoolStats, error) {
	var stats PoolStats
	err := d.sql.GetContext(ctx, &stats, "SELECT * FROM poolstats WHERE poolid = $1 ORDER BY created DESC FETCH NEXT 1 ROWS ONLY", poolID)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (d *DB) GetTotalPoolPayments(ctx context.Context, poolID string) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := d.sql.GetContext(ctx, &total, "SELECT sum(amount) FROM payments WHERE poolid = $1", poolID)
	if err != nil {
		return decimal.Zero, err
	}
	return total, nil
}

type SampleInterval string

const (
	IntervalHour SampleInterval = "hour"
	IntervalDay  SampleInterval = "day"
)

type SampleRange string

const (
	RangeHour  SampleRange = "hour"
	RangeDay   SampleRange = "day"
	RangeMonth SampleRange = "month"
)

var ErrInvalidSampleInterval = fmt.Errorf("invalid sample interval")

func (d *DB) GetPoolPerformanceBetween(ctx context.Context, poolID string, interval SampleInterval, start, end time.Time) ([]*AggregatedPoolStats, error) {
	var trunc string
	switch i := interval; i {
	case IntervalHour, IntervalDay:
		trunc = string(i)
	default:
		trunc = string(IntervalHour)
	}
	var stats []*AggregatedPoolStats
	err := d.sql.SelectContext(ctx, &stats, fmt.Sprintf(`
		SELECT date_trunc('%s', created) AS created,
		AVG(poolhashrate) AS poolhashrate, AVG(networkhashrate) AS networkhashrate, AVG(networkdifficulty) AS networkdifficulty,
		CAST(AVG(connectedminers) AS BIGINT) AS connectedminers
		FROM poolstats
		WHERE poolid = $1 AND created >= $2 AND created <= $3
		GROUP BY date_trunc('%s', created)
		ORDER BY created;
	`, trunc, trunc), poolID, start, end)
	return stats, err
}
