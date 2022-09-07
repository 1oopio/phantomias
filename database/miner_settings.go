package database

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type MinerSettingsSchema struct {
	PoolID           string
	Address          string
	PaymentThreshold decimal.Decimal
	Created          time.Time
	Updated          time.Time
}

type MinerSettings MinerSettingsSchema

func (d *DB) GetSettings(ctx context.Context, poolID, miner string) (MinerSettings, error) {
	var settings MinerSettings
	err := d.sql.GetContext(ctx, &settings, "SELECT * FROM miner_settings WHERE poolid = $1 AND address = $2", poolID, miner)
	if err != nil {
		return MinerSettings{}, fmt.Errorf("failed to get miner settings: %w", err)
	}
	return settings, nil
}

func (d *DB) UpdateSettings(ctx context.Context, settings MinerSettings) error {
	_, err := d.sql.ExecContext(ctx, `
		INSERT INTO miner_settings(poolid, address, paymentthreshold, created, updated)
			VALUES($1, $2, $3, now(), now())
			ON CONFLICT ON CONSTRAINT miner_settings_pkey DO UPDATE
			SET paymentthreshold = $3, updated = now()
				WHERE miner_settings.poolid = $1 
				AND miner_settings.address = $2
	`, settings.PoolID, settings.Address, settings.PaymentThreshold)
	if err != nil {
		return fmt.Errorf("failed to update miner settings: %w", err)
	}
	return nil
}
