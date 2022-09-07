package database

import (
	"time"

	"github.com/shopspring/decimal"
)

type BalanceSchema struct {
	PoolID  string
	Address string
	Amount  decimal.Decimal
	Created time.Time
	Updated time.Time
}
