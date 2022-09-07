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

func (d *DB) GetRecentyUsedIpAddresses(ctx context.Context, poolID, miner string) ([]string, error) {
	var ips []string
	err := d.sql.SelectContext(ctx, &ips, `
		SELECT DISTINCT s.ipaddress 
		FROM (
			SELECT * FROM shares
			WHERE poolid = $1 
			AND miner = $2 
			ORDER BY CREATED DESC 
			LIMIT 100
		) s;
	`, poolID, miner)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent ip addresses: %w", err)
	}
	return ips, nil
}
