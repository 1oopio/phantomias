package database

import (
	"context"
	"fmt"
)

type MinerSearchResult struct {
	Address string
	PoolID  string
}

func (d *DB) SearchMinerByAddress(ctx context.Context, miner string) ([]*MinerSearchResult, error) {
	searchResults := make([]*MinerSearchResult, 0)
	err := d.sql.SelectContext(ctx, &searchResults, `
	SELECT
		address,
		poolid
	FROM balances WHERE address LIKE '%' || $1 || '%'
	UNION
	SELECT
		DISTINCT miner AS address,
		poolid
	FROM shares
	WHERE
		miner LIKE '%' || $1 || '%' AND
		created > (NOW() - INTERVAL '8 minutes')
	group by
		miner,
		poolid;
	`, miner)
	if err != nil {
		return nil, fmt.Errorf("failed to search miner by address: %w", err)
	}
	return searchResults, nil
}
