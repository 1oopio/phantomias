package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type BlockSchema struct {
	ID                          int64 `json:"-"`
	PoolID                      string
	BlockHeight                 int64
	NetworkDifficulty           float64
	Status                      string
	Type                        *string
	ConfirmationProgress        float64
	Effort                      *float64
	TransactionConfirmationData string
	Miner                       string
	Reward                      decimal.Decimal
	Source                      string
	Hash                        *string
	Created                     time.Time
}

type Block BlockSchema

func (d *DB) GetPoolBlockCount(ctx context.Context, poolID string) (uint, error) {
	var count uint
	err := d.sql.GetContext(ctx, &count, "SELECT COUNT(*) FROM blocks WHERE poolid = $1", poolID)
	return count, err
}

func (d *DB) GetLastPoolBlockTime(ctx context.Context, poolID string) (time.Time, error) {
	var created time.Time
	err := d.sql.GetContext(ctx, &created, "SELECT created FROM blocks WHERE poolid = $1 ORDER BY created DESC LIMIT 1", poolID)
	return created, err
}

type BlockStatus string

const (
	BlockStatusConfirmed BlockStatus = "confirmed"
	BlockStatusPending   BlockStatus = "pending"
	BlockStatusOrphaned  BlockStatus = "orphaned"
)

func (d *DB) PageBlocks(ctx context.Context, poolID string, status []BlockStatus, page int, pageSize int) ([]*Block, error) {
	var (
		blocks []*Block
		s      strings.Builder
	)
	s.WriteString("SELECT id, poolid, blockheight, networkdifficulty, status, type, confirmationprogress, effort, transactionconfirmationdata, miner, reward, source, hash, created FROM blocks WHERE ")
	if poolID != "" {
		s.WriteString("poolid = $1 AND status = ANY($2) ORDER BY created DESC OFFSET $3 FETCH NEXT $4 ROWS ONLY;")
	} else {
		s.WriteString("status = ANY($1) ORDER BY created DESC OFFSET $2 FETCH NEXT $3 ROWS ONLY;")
	}
	var err error
	if poolID != "" {
		err = d.sql.SelectContext(ctx, &blocks, s.String(), poolID, status, page*pageSize, pageSize)
	} else {
		err = d.sql.SelectContext(ctx, &blocks, s.String(), status, page*pageSize, pageSize)
	}
	return blocks, err
}

func (d *DB) GetPoolEffort(ctx context.Context, poolID string, blocksCount int) (*float32, error) {
	var effort *float32
	err := d.sql.GetContext(ctx, &effort, `
	SELECT avg(effort) FROM (
		SELECT effort FROM blocks WHERE poolid = $1 AND effort IS NOT NULL ORDER BY created DESC FETCH NEXT $2 ROWS ONLY
	) as x;`, poolID, blocksCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool effort: %w", err)
	}
	return effort, nil
}
