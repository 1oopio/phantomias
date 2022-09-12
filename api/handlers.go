package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/go-miningcore-client"
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

	prices := s.price.GetPrices(strings.ToLower(p.Name))
	if prices != nil {
		priceRes := make(map[string]Price)
		for _, p := range prices {
			priceRes[p.VSCurrency] = Price{p.Price, p.PriceChangePercentage24H}
		}
		pool.Prices = priceRes
	}
	return &pool, nil
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
	topMinersRangeString := c.Query("topMinersRange", "1")
	topMinersRange, err := strconv.Atoi(topMinersRangeString)
	if err != nil {
		return handleAPIError(c, http.StatusBadRequest, err)
	}
	if topMinersRange < 1 || topMinersRange > 24 {
		topMinersRange = 1
	}

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

	prices := s.price.GetPrices(strings.ToLower(res.Result.Name))
	if prices != nil {
		priceRes := make(map[string]Price)
		for _, p := range prices {
			priceRes[p.VSCurrency] = Price{p.Price, p.PriceChangePercentage24H}
		}
		res.Result.Prices = priceRes
	}
	return c.JSON(res)
}

func poolPortsToAPIPoolPorts(p map[string]miningcore.PoolEndpoint) map[string]*PoolEndpoint {
	ports := make(map[string]*PoolEndpoint)
	for k, v := range p {
		ports[k] = poolPortToAPIPoolPort(v)
	}
	return ports
}

func poolPortToAPIPoolPort(p miningcore.PoolEndpoint) *PoolEndpoint {
	e := &PoolEndpoint{
		ListenAddress: p.ListenAddress,
		Name:          p.Name,
		Difficulty:    p.Difficulty,
		TLS:           p.TLS,
		TLSAuto:       p.TLSAuto,
	}
	if p.VarDiff != nil {
		vd := VarDiffConfig(*p.VarDiff)
		e.VarDiff = &vd
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
		Effort:                      b.Effort,
		TransactionConfirmationData: b.TransactionConfirmationData,
		Reward:                      b.Reward,
		InfoLink:                    sprintfOrEmpty(p.BlockLink, b.BlockHeight),
		Hash:                        b.Hash,
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
		AddressInfoLink:             sprintfOrEmpty(p.AddressLink, pmt.Address),
		Amount:                      pmt.Amount,
		TransactionConfirmationData: pmt.TransactionConfirmationData,
		TransactionInfoLink:         sprintfOrEmpty(p.TxLink, pmt.TransactionConfirmationData),
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

// @Summary Get a list of all miners
// @Description Get a list of all miners from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param page query int false "Page (default=0)"
// @Param pageSize query int false "PageSize (default=15)"
// @Success 200 {object} api.MinersRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners [get]
func (s *Server) getMinersHandler(c *fiber.Ctx) error {
	var miners []*MinerSimple
	code, err := s.mc.UnmarshalMiners(c.Context(), c.Params("id"), &miners, handlePaginationQueries(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(&MinersRes{
		Meta: &Meta{
			Success: true,
		},
		Result: miners,
	})
}

// @Summary Get a miner
// @Description Get a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param page query int false "Page (default=0)"
// @Param pageSize query int false "PageSize (default=15)"
// @Success 200 {object} api.MinerRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr} [get]
func (s *Server) getMinerHandler(c *fiber.Ctx) error {
	var miner Miner
	code, err := s.mc.UnmarshalMiner(c.Context(), c.Params("id"), c.Params("miner_addr"), &miner, handlePerformanceModeQuery(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(&MinerRes{
		Meta: &Meta{
			Success: true,
		},
		Result: &miner,
	})
}

func handlePerformanceModeQuery(c *fiber.Ctx) map[string]string {
	pm := c.Query("perfMode")
	if pm == "" {
		return nil
	}
	return map[string]string{
		"perfMode": pm,
	}
}

// @Summary Get payments
// @Description Get a list of payments from a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param page query int false "Page (default=0)"
// @Param pageSize query int false "PageSize (default=15)"
// @Success 200 {object} api.PaymentsRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/payments [get]
func (s *Server) getMinerPaymentsHandler(c *fiber.Ctx) error {
	var payments PaymentsRes
	code, err := s.mc.UnmarshalMinerPayments(c.Context(), c.Params("id"), c.Params("miner_addr"), &payments, handlePaginationQueries(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(payments)
}

// @Summary Get balance changes
// @Description Get a list of balance changes from a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param page query int false "Page (default=0)"
// @Param pageSize query int false "PageSize (default=15)"
// @Success 200 {object} api.BalanceChangesRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/balancechanges [get]
func (s *Server) getMinerBalanceChangesHandler(c *fiber.Ctx) error {
	var balanceChanges BalanceChangesRes
	code, err := s.mc.UnmarshalMinerBalanceChanges(c.Context(), c.Params("id"), c.Params("miner_addr"), &balanceChanges, handlePaginationQueries(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(balanceChanges)
}

// @Summary Get daily earnings
// @Description Get a list of daily earnings from a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param page query int false "Page (default=0)"
// @Param pageSize query int false "PageSize (default=15)"
// @Success 200 {object} api.DailyEarningRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/earnings/daily [get]
func (s *Server) getMinerDailyEarningsHandler(c *fiber.Ctx) error {
	var dailyEarnings DailyEarningRes
	code, err := s.mc.UnmarshalMinerDailyEarnings(c.Context(), c.Params("id"), c.Params("miner_addr"), &dailyEarnings, handlePaginationQueries(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(dailyEarnings)
}

// Maybe we should put a higher rate limit on this endpoint.
// It the call seems rather slow (in my quick and dirty test) and seems to return a lot of data.
// Should we implement a paged v2 endpoint on miningcore for this?

// @Summary Get performance
// @Description Get a list of performance samples from a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param sampleRange query string Daily "Specify the sample range (default=Daily)"
// @Success 200 {object} api.MinerPerformanceRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/performance [get]
func (s *Server) getMinerPerformanceHandler(c *fiber.Ctx) error {
	var performance []*WorkerStats
	code, err := s.mc.UnmarshalMinerPerformance(c.Context(), c.Params("id"), c.Params("miner_addr"), &performance, handleSampleRangeQuery(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(&MinerPerformanceRes{
		Meta: &Meta{
			Success: true,
		},
		Result: performance,
	})
}

func handleSampleRangeQuery(c *fiber.Ctx) map[string]string {
	pm := c.Query("sampleRange")
	if pm == "" {
		return nil
	}
	return map[string]string{
		"sampleRange": pm,
	}
}

// @Summary Get settings
// @Description Get the settings from a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Success 200 {object} api.MinerSettingsRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/settings [get]
func (s *Server) getMinerSettingsHandler(c *fiber.Ctx) error {
	var settings MinerSettings
	code, err := s.mc.UnmarshalMinerSettings(c.Context(), c.Params("id"), c.Params("miner_addr"), &settings)
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(&MinerSettingsRes{
		Meta: &Meta{
			Success: true,
		},
		Result: &settings,
	})
}

// @Summary Update settings
// @Description Update the settings from a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param settings body api.MinerSettingsReq true "Updated settings incl. the IP of the highest worker"
// @Success 200 {object} api.MinerSettingsRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/settings [post]
func (s *Server) postMinerSettingsHandler(c *fiber.Ctx) error {
	var req MinerSettingsReq
	if err := c.BodyParser(&req); err != nil {
		return handleAPIError(c, http.StatusBadRequest, err)
	}
	var settings MinerSettings
	code, err := s.mc.UnmarshalPostMinerSettings(c.Context(), c.Params("id"), c.Params("miner_addr"), &req, &settings)
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(&MinerSettingsRes{
		Meta: &Meta{
			Success: true,
		},
		Result: &settings,
	})
}
