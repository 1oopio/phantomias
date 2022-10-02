package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/phantomias/config"
	"github.com/stratumfarm/phantomias/database"
	"github.com/stratumfarm/phantomias/utils"
)

// @Summary Get all pools
// @Description Get a list of all available pools
// @Tags Pools
// @Produce  json
// @Success 200 {array} api.PoolsRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools [get]
func (s *Server) getPoolsHandler(c *fiber.Ctx) error {
	result := make([]*Pool, 0)
	for _, p := range s.pools {
		if !p.Enabled {
			continue
		}
		pool, err := s.gatherPoolStats(c.Context(), p)
		if err != nil {
			return handleAPIError(c, http.StatusInternalServerError, err)
		}
		result = append(result, pool)
	}

	return c.JSON(&PoolsRes{
		Meta: &Meta{
			Success: true,
		},
		Result: result,
	})
}

func (s *Server) gatherPoolStats(ctx context.Context, p *config.Pool) (*Pool, error) {
	var pool Pool
	stats, err := s.db.GetLastPoolStats(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	pool.ID = stats.PoolID
	pool.Algorithm = p.Algorithm
	pool.Name = p.Name
	pool.Coin = p.Coin
	pool.Fee = p.Fee
	pool.FeeType = p.FeeType
	pool.Miners = stats.ConnectedMiners
	pool.Hashrate = stats.PoolHashrate
	pool.BlockHeight = stats.BlockHeight
	pool.NetworkHashrate = stats.NetworkHashrate
	pool.NetworkDifficulty = stats.NetworkDifficulty

	pool.Prices = s.getPrices(p.Name)

	return &pool, nil
}

func (s Server) getPrices(name string) (priceRes map[string]Price) {
	priceRes = make(map[string]Price)
	prices := s.price.GetPrices(strings.ToLower(name))
	if prices != nil {
		for _, p := range prices {
			priceRes[p.VSCurrency] = Price{p.Price, p.PriceChangePercentage24H}
		}
	}
	return
}

// @Summary Get a pool
// @Description Get a specific pool
// @Tags Pools
// @Produce  json
// @Param pool_id path string true "ID of the pool"
// @Success 200 {object} api.PoolExtendedRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id} [get]
func (s *Server) getPoolHandler(c *fiber.Ctx) error {
	topMinersRange := getTopMinersRange(c)

	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}

	poolStats, err := s.gatherPoolStats(c.Context(), poolCfg)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	from := time.Now().Add(-time.Duration(topMinersRange) * time.Hour)
	minersByHashrate, err := s.db.PagePoolMinersByHashrate(c.Context(), c.Params("id"), from, 0, 15)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	poolExtended := PoolExtended{Pool: poolStats, TopMiners: minersByHashrate}

	totalPaid, err := s.db.GetTotalPoolPayments(c.Context(), c.Params("id"))
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	poolExtended.TotalPayments = totalPaid.InexactFloat64()

	totalBlocks, err := s.db.GetPoolBlockCount(c.Context(), c.Params("id"))
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	poolExtended.TotalBlocksFound = totalBlocks

	lastPoolBlockTime, err := s.db.GetLastPoolBlockTime(c.Context(), c.Params("id"))
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	poolExtended.LastBlockFoundTime = lastPoolBlockTime

	res := &PoolExtendedRes{
		Meta: &Meta{
			Success: true,
		},
		Result: &poolExtended,
	}
	res.Result.Ports = cfgPortsToAPIPoolPorts(poolCfg.Ports)
	res.Result.Prices = s.getPrices(res.Result.Name)
	return c.JSON(res)
}

func cfgPortsToAPIPoolPorts(p map[string]config.Port) map[string]*PoolEndpoint {
	ports := make(map[string]*PoolEndpoint)
	for k, v := range p {
		ports[k] = cfgPortToAPIPoolPort(v)
	}
	return ports
}

func cfgPortToAPIPoolPort(p config.Port) *PoolEndpoint {
	e := &PoolEndpoint{
		Difficulty: p.Difficulty,
		VarDiff:    true,
		TLS:        p.TLS,
		TLSAuto:    p.TLSAuto,
	}
	return e
}

type BlocksParams struct {
	BlockStatus []database.BlockStatus `query:"blockStatus"`
}

// @Summary Get a list of blocks
// @Description Get a list of blocks from a specific pool
// @Tags Pools
// @Produce  json
// @Param pool_id path string true "ID of the pool"
// @Param page query int false "Page (default=0)"
// @Param pageSize query int false "PageSize (default=15)"
// @Success 200 {object} api.BlocksRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/blocks [get]
func (s *Server) getBlocksHandler(c *fiber.Ctx) error {
	params := new(BlocksParams)
	if err := c.QueryParser(params); err != nil {
		// TODO: log error
		return handleAPIError(c, http.StatusBadRequest, fmt.Errorf("failed to parse params"))
	}
	if len(params.BlockStatus) == 0 {
		params.BlockStatus = []database.BlockStatus{database.BlockStatusConfirmed, database.BlockStatusOrphaned, database.BlockStatusPending}
	}

	pool := getPoolCfgByID(c.Params("id"), s.pools)
	if pool == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}

	pageCount, err := s.db.GetPoolBlockCount(c.Context(), pool.ID)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	page, pageSize := getPageParams(c)
	pageCount = uint(math.Floor(float64(pageCount) / float64(pageSize)))

	blocks, err := s.db.PageBlocks(c.Context(), c.Params("id"), params.BlockStatus, page, pageSize)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	res := &BlocksRes{
		Meta: &Meta{
			Success:   true,
			PageCount: pageCount,
		},
		Result: dbBlocksToAPIBlocks(pool, blocks),
	}
	return c.JSON(res)
}

func dbBlocksToAPIBlocks(p *config.Pool, b []*database.Block) []*Block {
	blocks := make([]*Block, len(b))
	for i, block := range b {
		blocks[i] = dbBlockToAPIBlock(p, block)
	}
	return blocks
}

func dbBlockToAPIBlock(p *config.Pool, b *database.Block) *Block {
	return &Block{
		PoolID:                      b.PoolID,
		BlockHeight:                 b.BlockHeight,
		NetworkDifficulty:           b.NetworkDifficulty,
		Status:                      b.Status,
		ConfirmationProgress:        b.ConfirmationProgress,
		Effort:                      utils.ValueOrZero(b.Effort),
		TransactionConfirmationData: b.TransactionConfirmationData,
		Reward:                      b.Reward.InexactFloat64(),
		InfoLink:                    getBlockLink(p, b),
		Hash:                        utils.ValueOrZero(b.Hash),
		Miner:                       b.Miner,
		Source:                      b.Source,
		Created:                     b.Created,
	}
}

// @Summary Get a list of payments
// @Description Get a list of payments from a specific pool
// @Tags Pools
// @Produce  json
// @Param pool_id path string true "ID of the pool"
// @Param page query int false "Page (default=0)"
// @Param pageSize query int false "PageSize (default=15)"
// @Success 200 {object} api.PaymentsRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/payments [get]
func (s *Server) getPaymentsHandler(c *fiber.Ctx) error {
	pool := getPoolCfgByID(c.Params("id"), s.pools)
	if pool == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}

	pageCount, err := s.db.GetPaymentsCount(c.Context(), pool.ID, "")
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	page, pageSize := getPageParams(c)
	pageCount = uint(math.Floor(float64(pageCount) / float64(pageSize)))

	payments, err := s.db.PagePayments(c.Context(), pool.ID, "", page, pageSize)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	res := &PaymentsRes{
		Meta: &Meta{
			Success:   true,
			PageCount: pageCount,
		},
		Result: dbPaymentsToAPIPayments(pool, payments),
	}
	return c.JSON(res)
}

func dbPaymentsToAPIPayments(p *config.Pool, pmts []*database.Payment) []*Payment {
	payments := make([]*Payment, len(pmts))
	for i, pmt := range pmts {
		payments[i] = dbPaymentToAPIPayment(p, pmt)
	}
	return payments
}

func dbPaymentToAPIPayment(p *config.Pool, pmt *database.Payment) *Payment {
	return &Payment{
		Coin:                        pmt.Coin,
		Address:                     pmt.Address,
		AddressInfoLink:             getAddressLink(p.AddressLink, pmt.Address),
		Amount:                      pmt.Amount.InexactFloat64(),
		TransactionConfirmationData: pmt.TransactionConfirmationData,
		TransactionInfoLink:         getTXLink(p.TxLink, pmt.TransactionConfirmationData),
		Created:                     pmt.Created,
	}
}

// @Summary Get a list of performance samples
// @Description Get a list of performance samples from a specific pool
// @Tags Pools
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param i query string false "sample interval (default=Hour)"
// @Param r query string false "sample range (default=Day)"
// @Success 200 {object} api.PoolPerformanceRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/performance [get]
func (s *Server) getPoolPerformanceHandler(c *fiber.Ctx) error {
	performanceRange := c.Query("r", string(database.RangeDay))
	performanceInterval := c.Query("i", string(database.IntervalHour))

	pool := getPoolCfgByID(c.Params("id"), s.pools)
	if pool == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}

	end := time.Now()
	var start time.Time

	switch database.SampleRange(performanceRange) {
	case database.RangeHour:
		start = end.Add(-1 * time.Hour)
	case database.RangeDay:
		start = end.Add(-24 * time.Hour)
	case database.RangeMonth:
		start = end.Add(-30 * 24 * time.Hour)
	default:
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidRange)
	}

	stats, err := s.db.GetPoolPerformanceBetween(c.Context(), pool.ID, database.SampleInterval(performanceInterval), start, end)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	return c.JSON(&PoolPerformanceRes{
		Meta: &Meta{
			Success: true,
		},
		Result: dbPoolPerformanceToAPIPerformance(stats),
	})
}

func dbPoolPerformanceToAPIPerformance(stats []*database.AggregatedPoolStats) []*PoolPerformance {
	perfStats := make([]*PoolPerformance, len(stats))
	for i, stat := range stats {
		s := PoolPerformance(*stat)
		perfStats[i] = &s
	}
	return perfStats
}
