package api

import (
	"time"
)

type Meta struct {
	PageCount uint `json:"pageCount"`
	Success   bool `json:"success"`
}

type StatsRes struct {
	*Meta
	Result Stats `json:"result"`
}

type Stats struct {
	TotalMiners          int32   `json:"totalMiners"`
	TotalWorkers         int32   `json:"totalWorkers"`
	TotalSharesPerSecond float64 `json:"totalSharesPerSecond"`
	PaymentsToday        int32   `json:"paymentsToday"`
}

type MinerSearchRes struct {
	*Meta
	Result []MinerSearch `json:"result"`
}

type MinerSearch struct {
	Address string `json:"address"`
	PoolID  string `json:"poolId"`
	FeeType string `json:"feeType"`
}

type PoolsRes struct {
	*Meta
	Result []*Pool `json:"result"`
}

type Pool struct {
	Coin              string           `json:"coin"`
	ID                string           `json:"id"`
	Algorithm         string           `json:"algorithm"`
	Name              string           `json:"name"`
	Hashrate          float64          `json:"hashrate"`
	Miners            int32            `json:"miners"`
	Workers           int32            `json:"workers"`
	Fee               float64          `json:"fee"`
	FeeType           string           `json:"feeType"`
	BlockHeight       int64            `json:"blockHeight"`
	NetworkHashrate   float64          `json:"networkHashrate"`
	NetworkDifficulty float64          `json:"networkDifficulty"`
	Prices            map[string]Price `json:"prices"`
}

type PoolExtendedRes struct {
	*Meta
	Result *PoolExtended `json:"result"`
}

type PoolExtended struct {
	*Pool
	Type               string                   `json:"type"`
	Address            string                   `json:"address"`
	MinPayout          float64                  `json:"minPayout"`
	Ports              map[string]*PoolEndpoint `json:"ports"`
	TotalBlocksFound   uint                     `json:"totalBlocksFound"`
	TotalPayments      float64                  `json:"totalPayments"`
	LastBlockFoundTime time.Time                `json:"lastBlockFoundTime"`
	AverageEffort      *float32                 `json:"averageEffort"`
	Effort             float32                  `json:"effort"`
}

type Price struct {
	Price                    float64 `json:"price"`
	PriceChangePercentage24H float64 `json:"priceChangePercentage24H"`
}

type PoolEndpoint struct {
	Difficulty float64 `json:"difficulty"`
	VarDiff    bool    `json:"varDiff"`
	TLS        bool    `json:"tls"`
	TLSAuto    bool    `json:"tlsAuto"`
}

type BlocksRes struct {
	*Meta
	Result []*Block `json:"result"`
}

type Block struct {
	PoolID                      string    `json:"poolId"`
	BlockHeight                 int64     `json:"blockHeight"`
	NetworkDifficulty           float64   `json:"networkDifficulty"`
	Status                      string    `json:"status"`
	ConfirmationProgress        float64   `json:"confirmationProgress"`
	Effort                      float64   `json:"effort"`
	TransactionConfirmationData string    `json:"transactionConfirmationData"`
	Reward                      float64   `json:"reward"`
	InfoLink                    string    `json:"infoLink,omitempty"`
	Hash                        string    `json:"hash"`
	Miner                       string    `json:"miner"`
	Source                      string    `json:"source"`
	Created                     time.Time `json:"created"`
}

type PaymentsRes struct {
	*Meta
	Result []*Payment `json:"result"`
}

type Payment struct {
	Coin                        string    `json:"coin"`
	Address                     string    `json:"address"`
	AddressInfoLink             string    `json:"addressInfoLink,omitempty"`
	Amount                      float64   `json:"amount"`
	TransactionConfirmationData string    `json:"transactionConfirmationData"`
	TransactionInfoLink         string    `json:"transactionInfoLink,omitempty"`
	Created                     time.Time `json:"created"`
}

type PoolPerformanceRes struct {
	*Meta
	Result []*PoolPerformance `json:"result"`
}

type PoolPerformance struct {
	PoolHashrate      float64   `json:"poolHashrate"`
	ConnectedMiners   int       `json:"connectedMiners"`
	NetworkHashrate   float64   `json:"networkHashrate"`
	NetworkDifficulty float64   `json:"networkDifficulty"`
	Created           time.Time `json:"created"`
}

type MinersRes struct {
	*Meta
	Result []MinerSimple `json:"result"`
}

type MinerSimple struct {
	Miner           string  `json:"miner"`
	Hashrate        float64 `json:"hashrate"`
	SharesPerSecond float64 `json:"sharesPerSecond"`
}

type MinerRes struct {
	*Meta
	Result *Miner `json:"result"`
}

type Miner struct {
	PendingShares   float64                          `json:"pendingShares"`
	PendingBalance  float64                          `json:"pendingBalance"`
	TotalPaid       float64                          `json:"totalPaid"`
	TodayPaid       float64                          `json:"todayPaid"`
	LastPayment     *time.Time                       `json:"lastPayment"`
	LastPaymentLink string                           `json:"lastPaymentLink"`
	Performance     *WorkerPerformanceStatsContainer `json:"performance"`
	Prices          map[string]Price                 `json:"prices"`
	Coin            string                           `json:"coin"`
}

type WorkerPerformanceStatsContainer struct {
	Created time.Time                          `json:"created"`
	Workers map[string]*WorkerPerformanceStats `json:"workers"`
}

type WorkerPerformanceStats struct {
	Hashrate         *float64 `json:"hashrate"`
	ReportedHashrate *float64 `json:"reportedHashrate"`
	SharesPerSecond  *float64 `json:"sharesPerSecond"`
}

type PerformanceStats struct {
	Created          time.Time `json:"created"`
	Hashrate         float64   `json:"hashrate"`
	ReportedHashrate float64   `json:"reportedHashrate"`
	SharesPerSecond  float64   `json:"sharesPerSecond"`
	WorkersOnline    uint      `json:"workersOnline,omitempty"`
}

type BalanceChangesRes struct {
	*Meta
	Result []*BalanceChange `json:"result"`
}

type BalanceChange struct {
	PoolID  string    `json:"poolId"`
	Address string    `json:"address"`
	Amount  float64   `json:"amount"`
	Usage   string    `json:"usage"`
	Created time.Time `json:"created"`
}

type DailyEarning struct {
	Amount float64   `json:"amount"`
	Date   time.Time `json:"date"`
}

type DailyEarningRes struct {
	*Meta
	Result []*DailyEarning `json:"result"`
}

type MinerPerformanceRes struct {
	*Meta
	Result []*PerformanceStats `json:"result"`
}

type MinerSettingsRes struct {
	*Meta
	Result *MinerSettings `json:"result"`
}

type MinerSettings struct {
	PaymentThreshold float64 `json:"paymentThreshold"`
}

type MinerSettingsReq struct {
	IPAddress string         `json:"ipAddress"`
	Settings  *MinerSettings `json:"settings"`
}

type WorkerPerformanceRes struct {
	*Meta
	Result []*PerformanceStats `json:"result"`
}

type Worker struct {
	Hashrate        float64 `json:"hashrate"`
	SharesPerSecond float64 `json:"sharesPerSecond"`
}

type WorkerRes struct {
	*Meta
	Result *Worker `json:"result"`
}

type TopMinersRes struct {
	*Meta
	Result []*TopMiner `json:"result"`
}

type TopMiner struct {
	Miner     string     `json:"miner"`
	Hashrate  float64    `json:"hashrate"`
	Workers   int        `json:"workers"`
	TotalPaid float64    `json:"totalPaid"`
	Joined    *time.Time `json:"joined"`
}
