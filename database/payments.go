package database

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type PaymentSchema struct {
	ID                          int64
	PoolID                      string
	Coin                        string
	Address                     string
	Amount                      decimal.Decimal
	TransactionConfirmationData string
	Created                     time.Time
}

type Payment PaymentSchema

type AmountByDate struct {
	Amount decimal.Decimal
	Date   time.Time
}

func (d *DB) PagePayments(ctx context.Context, poolID, address string, page int, pageSize int) ([]*Payment, error) {
	var payments []*Payment
	var s strings.Builder

	s.WriteString("SELECT * FROM payments WHERE poolid = $1")
	if address != "" {
		s.WriteString(" AND address = $2 ORDER BY created DESC OFFSET $3 FETCH NEXT $4 ROWS ONLY;")
	} else {
		s.WriteString(" ORDER BY created DESC OFFSET $2 FETCH NEXT $3 ROWS ONLY;")
	}
	var err error
	if address != "" {
		err = d.sql.SelectContext(ctx, &payments, s.String(), poolID, address, page*pageSize, pageSize)
	} else {
		err = d.sql.SelectContext(ctx, &payments, s.String(), poolID, page*pageSize, pageSize)
	}
	return payments, err
}

func (d *DB) GetPaymentsCount(ctx context.Context, poolID, address string) (uint, error) {
	var count uint
	var err error
	if address != "" {
		err = d.sql.GetContext(ctx, &count, "SELECT COUNT(*) FROM payments WHERE poolid = $1 AND address = $2", poolID, address)
	} else {
		err = d.sql.GetContext(ctx, &count, "SELECT COUNT(*) FROM payments WHERE poolid = $1", poolID)
	}
	return count, err
}

func (d *DB) GetMinerPaymentsByDayCount(ctx context.Context, poolID, miner string) (uint, error) {
	var count uint
	err := d.sql.GetContext(ctx, &count, `
	SELECT COUNT(*) FROM (
		SELECT SUM(amount) AS amount, date_trunc('day', created) AS date FROM payments WHERE poolid = $1
		AND address = $2 GROUP BY date ORDER BY date DESC) s;`, poolID, miner)
	return count, err
}

func (d *DB) PageMinerPaymentsByDay(ctx context.Context, poolID, address string, page, pageSize int) ([]*AmountByDate, error) {
	var payments []*AmountByDate
	err := d.sql.SelectContext(ctx, &payments, `
	SELECT SUM(amount) AS amount, date_trunc('day', created) AS date FROM payments WHERE poolid = $1
		AND address = $2 GROUP BY date ORDER BY date DESC OFFSET $3 FETCH NEXT $4 ROWS ONLY;`, poolID, address, page*pageSize, pageSize)
	return payments, err
}
