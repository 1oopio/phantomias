package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type BalanceChangeSchema struct {
	ID      int64
	PoolID  string
	Address string
	Amount  decimal.Decimal
	Usage   string
	Tags    *string
	Created time.Time
}

type BalanceChange struct {
	PoolID  string
	Address string
	Amount  decimal.Decimal
	Usage   string
	Created time.Time
}

func (d *DB) PageBalanceChanges(ctx context.Context, poolID, miner string, page, pageSize int) ([]BalanceChange, error) {
	var rawBalanceChanges []BalanceChangeSchema
	err := d.sql.SelectContext(ctx, &rawBalanceChanges, `
	SELECT * FROM balance_changes WHERE poolid = $1 AND address = $2
		ORDER BY created DESC OFFSET $3 FETCH NEXT $4 ROWS ONLY;
	`, poolID, miner, page*pageSize, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to select balance_changes: %v", err)
	}
	balanceChanges := make([]BalanceChange, len(rawBalanceChanges))
	for i, rawBalanceChange := range rawBalanceChanges {
		balanceChanges[i] = BalanceChange{
			PoolID:  rawBalanceChange.PoolID,
			Address: rawBalanceChange.Address,
			Amount:  rawBalanceChange.Amount,
			Usage:   rawBalanceChange.Usage,
			Created: rawBalanceChange.Created,
		}
	}
	return balanceChanges, nil
}

func (d *DB) GetBalanceChangesCount(ctx context.Context, poolID, miner string) (uint, error) {
	var s strings.Builder
	s.WriteString("SELECT COUNT(*) FROM balance_changes WHERE poolid = $1")
	if miner != "" {
		s.WriteString(" AND address = $2")
	}
	var count uint
	if miner != "" {
		err := d.sql.GetContext(ctx, &count, s.String(), poolID, miner)
		if err != nil {
			return 0, fmt.Errorf("failed to select balance_changes count: %v", err)
		}
	} else {
		err := d.sql.GetContext(ctx, &count, s.String(), poolID)
		if err != nil {
			return 0, fmt.Errorf("failed to select balance_changes count: %v", err)
		}
	}
	return count, nil
}
