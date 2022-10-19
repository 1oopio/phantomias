package database

import (
	"context"
	"fmt"
	"time"
)

type Share struct {
	PoolID            string
	BlockHeight       int64
	Difficulty        float64
	NetworkDifficulty float64
	Miner             string
	Worker            string
	UserAgent         string
	IPAddress         string
	Source            string
	Created           time.Time
}

func (d *DB) GetRecentyUsedIPAddresses(ctx context.Context, poolID, miner string) ([]string, error) {
	var ips []string
	err := d.sql.SelectContext(ctx, &ips, `
		SELECT DISTINCT s.ipaddress 
		FROM (
			SELECT
				poolid,
				blockheight,
				difficulty,
				networkdifficulty,
				miner,
				worker,
				useragent,
				ipaddress,
				source,
				created
			FROM shares
			WHERE 
				poolid = $1 AND 
				miner = $2 
			ORDER BY 
				created DESC 
			LIMIT 100
		) s;
	`, poolID, miner)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent ip addresses: %w", err)
	}
	return ips, nil
}

func (d *DB) GetEffortBetweenCreated(ctx context.Context, poolID string, shareConst float64, start, end time.Time) (*float32, error) {
	var effort *float32
	err := d.sql.GetContext(ctx, &effort, `
	SELECT SUM((difficulty*$1)/(networkdifficulty)) FROM shares WHERE poolid = $2 AND created > $3 AND created < $4;
	`, shareConst, poolID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get effort between created: %w", err)
	}
	return effort, nil
}
