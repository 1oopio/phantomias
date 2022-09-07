package api

type Meta struct {
	PageCount int64 `json:"pageCount"`
	Success   bool  `json:"success"`
}

type PoolsRes struct {
	*Meta
	Result []*Pool `json:"result"`
}

type Pool struct {
	Coin            string           `json:"coin"`
	ID              string           `json:"id"`
	Algorithm       string           `json:"algorithm"`
	Name            string           `json:"name"`
	Hashrate        int64            `json:"hashrate"`
	Miners          int32            `json:"miners"`
	Fee             float64          `json:"fee"`
	FeeType         string           `json:"feeType"`
	BlockHeight     int64            `json:"blockHeight"`
	NetworkHashrate float64          `json:"networkHashrate"`
	Prices          map[string]Price `json:"prices"`
}

type PoolExtendedRes struct {
	*Meta
	Result *PoolExtended `json:"result"`
}

type PoolExtended struct {
	*Pool
	Ports              map[string]*PoolEndpoint `json:"ports"`
	TotalBlocksFound   int32                    `json:"totalBlocksFound"`
	TotalPayments      float64                  `json:"totalPayments"`
	LastBlockFoundTime int64                    `json:"lastBlockFoundTime"`
}

type Price struct {
	Price                    float64 `json:"price"`
	PriceChangePercentage24H float64 `json:"priceChangePercentage24H"`
}

type PoolEndpoint struct {
	ListenAddress string         `json:"listenAddress"`
	Name          string         `json:"name"`
	Difficulty    float64        `json:"difficulty"`
	VarDiff       *VarDiffConfig `json:"varDiff"`
	TLS           bool           `json:"tls"`
	TLSAuto       bool           `json:"tlsAuto"`
}

type VarDiffConfig struct {
	MinDiff         float64 `json:"minDiff"`
	MaxDiff         float64 `json:"maxDiff"`
	MaxDelta        float64 `json:"maxDelta"`
	TargetTime      float64 `json:"targetTime"`
	RetargetTime    float64 `json:"retargetTime"`
	VariancePercent float64 `json:"variancePercent"`
}

type BlocksRes struct {
	*Meta
	Result []*Block `json:"result"`
}

type Block struct {
	PoolID                      string  `json:"poolId"`
	BlockHeight                 int64   `json:"blockHeight"`
	NetworkDifficulty           float64 `json:"networkDifficulty"`
	Status                      string  `json:"status"`
	Type                        string  `json:"type"`
	ConfirmationProgress        float64 `json:"confirmationProgress"`
	Effort                      float64 `json:"effort"`
	TransactionConfirmationData string  `json:"transactionConfirmationData"`
	Reward                      float64 `json:"reward"`
	InfoLink                    string  `json:"infoLink"`
	Hash                        string  `json:"hash"`
	Miner                       string  `json:"miner"`
	Source                      string  `json:"source"`
	Created                     string  `json:"created"`
}

type PaymentsRes struct {
	*Meta
	Result []*Payment `json:"result"`
}

type Payment struct {
	Coin                        string  `json:"coin"`
	Address                     string  `json:"address"`
	AddressInfoLink             string  `json:"addressInfoLink"`
	Amount                      float64 `json:"amount"`
	TransactionConfirmationData string  `json:"transactionConfirmationData"`
	TransactionInfoLink         string  `json:"transactionInfoLink"`
	Created                     string  `json:"created"`
}

type PoolPerformanceRes struct {
	*Meta
	Result []*PoolPerformance `json:"result"`
}

type PoolPerformance struct {
	PoolHashrate         float64 `json:"poolHashrate"`
	ConnectedMiners      int32   `json:"connectedMiners"`
	ValidSharesPerSecond int32   `json:"validSharesPerSecond"`
	NetworkHashrate      float64 `json:"networkHashrate"`
	NetworkDifficulty    float64 `json:"networkDifficulty"`
	Created              string  `json:"created"`
}

type MinersRes struct {
	*Meta
	Result []*MinerSimple `json:"result"`
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
	PendingShares      int64          `json:"pendingShares"`
	PendingBalance     float64        `json:"pendingBalance"`
	TotalPaid          float64        `json:"totalPaid"`
	TodayPaid          float64        `json:"todayPaid"`
	LastPayment        string         `json:"lastPayment"`
	LastPaymentLink    string         `json:"lastPaymentLink"`
	Performance        *WorkerStats   `json:"performance"`
	PerformanceSamples []*WorkerStats `json:"performanceSamples"`
}

type WorkerStats struct {
	Created string                             `json:"created"`
	Workers map[string]*WorkerPerformanceStats `json:"workers"`
}

type WorkerPerformanceStats struct {
	Hashrate         float64 `json:"hashrate"`
	ReportedHashrate float64 `json:"reportedHashrate"`
	SharesPerSecond  float64 `json:"sharesPerSecond"`
}

type BalanceChangesRes struct {
	*Meta
	Result []*BalanceChange `json:"result"`
}

type BalanceChange struct {
	PoolID  string  `json:"poolId"`
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
	Usage   string  `json:"usage"`
	Created string  `json:"created"`
}

type DailyEarning struct {
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
}

type DailyEarningRes struct {
	*Meta
	Result []*DailyEarning `json:"result"`
}

type MinerPerformanceRes struct {
	*Meta
	Result []*WorkerStats `json:"result"`
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
