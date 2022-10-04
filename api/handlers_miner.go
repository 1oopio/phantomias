package api

import (
	"math"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/phantomias/database"
	"github.com/stratumfarm/phantomias/utils"
)

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
	topMinersRange := getTopMinersRange(c)

	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}

	from := time.Now().Add(-time.Duration(topMinersRange) * time.Hour)
	pageCount, err := s.db.GetMinersCount(c.Context(), poolCfg.ID, from)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	page, pageSize := getPageParams(c)
	pageCount = uint(math.Floor(float64(pageCount) / float64(pageSize)))

	minersByHashrate, err := s.db.PagePoolMinersByHashrate(c.Context(), poolCfg.ID, from, page, pageSize)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	res := &MinersRes{
		Meta: &Meta{
			Success:   true,
			PageCount: pageCount,
		},
		Result: dbMinersToAPIMiners(minersByHashrate),
	}
	return c.JSON(res)
}

func dbMinersToAPIMiners(miners []database.MinerPerformanceStats) []MinerSimple {
	apiMiners := make([]MinerSimple, len(miners))
	for i, miner := range miners {
		apiMiners[i] = MinerSimple(miner)
	}
	return apiMiners
}

// @Summary Get a miner
// @Description Get a specific miner from a specific pool
// @Tags Miners
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Success 200 {object} api.MinerRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr} [get]
func (s *Server) getMinerHandler(c *fiber.Ctx) error {
	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}
	addr := getMinerAddress(c, poolCfg)
	if addr == "" {
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}

	stats, err := s.db.GetMinerStats(c.Context(), poolCfg.ID, addr)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	if stats == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrNoStatsFound)
	}

	// TODO: multiply pendig shares with share multiplier

	miner := dbMinerStatsToAPIMiner(stats)
	if stats.LastPayment != nil && (stats.LastPayment != &database.Payment{}) {
		miner.LastPayment = &stats.LastPayment.Created
		miner.LastPaymentLink = getTXLink(poolCfg.TxLink, stats.LastPayment.TransactionConfirmationData)
	}
	miner.Prices = s.getPrices(poolCfg.Name)
	miner.Coin = poolCfg.Coin
	return c.JSON(&MinerRes{
		Meta: &Meta{
			Success: true,
		},
		Result: miner,
	})
}

func dbMinerStatsToAPIMiner(stats *database.MinerStats) *Miner {
	miner := &Miner{
		PendingShares:  utils.ValueOrZero(stats.PendingShares),
		PendingBalance: utils.ValueOrZero(stats.PendingBalance),
		TotalPaid:      utils.ValueOrZero(stats.TotalPaid),
		TodayPaid:      utils.ValueOrZero(stats.TodayPaid),
	}
	if stats.Performance != nil {
		workerStats := &WorkerPerformanceStatsContainer{
			Created: stats.Performance.Created,
			Workers: dbWorkersStatsToAPIWorkerStats(stats.Performance.Workers),
		}
		miner.Performance = workerStats
	}
	return miner
}

func dbWorkersStatsToAPIWorkerStats(stats map[string]*database.WorkerPerformanceStats) map[string]*WorkerPerformanceStats {
	workers := make(map[string]*WorkerPerformanceStats, len(stats))
	for addr, stat := range stats {
		workers[addr] = &WorkerPerformanceStats{
			Hashrate:         stat.Hashrate,
			ReportedHashrate: stat.ReportedHashrate,
			SharesPerSecond:  stat.SharesPerSecond,
		}
	}
	return workers
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
	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}
	addr := getMinerAddress(c, poolCfg)
	if addr == "" {
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}

	pageCount, err := s.db.GetPaymentsCount(c.Context(), poolCfg.ID, addr)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	page, pageSize := getPageParams(c)
	pageCount = uint(math.Floor(float64(pageCount) / float64(pageSize)))

	payments, err := s.db.PagePayments(c.Context(), poolCfg.ID, addr, page, pageSize)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	res := &PaymentsRes{
		Meta: &Meta{
			Success:   true,
			PageCount: pageCount,
		},
		Result: dbPaymentsToAPIPayments(poolCfg, payments),
	}
	return c.JSON(res)
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
	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}
	addr := getMinerAddress(c, poolCfg)
	if addr == "" {
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}

	pageCount, err := s.db.GetBalanceChangesCount(c.Context(), poolCfg.ID, addr)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	page, pageSize := getPageParams(c)
	pageCount = uint(math.Floor(float64(pageCount) / float64(pageSize)))

	balanceChanges, err := s.db.PageBalanceChanges(c.Context(), poolCfg.ID, addr, page, pageSize)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	res := BalanceChangesRes{
		Meta: &Meta{
			Success:   true,
			PageCount: pageCount,
		},
		Result: dbBalanceChangesToAPI(balanceChanges),
	}
	return c.JSON(res)
}

func dbBalanceChangesToAPI(balanceChanges []*database.BalanceChange) []*BalanceChange {
	res := make([]*BalanceChange, len(balanceChanges))
	for i, bc := range balanceChanges {
		res[i] = &BalanceChange{
			PoolID:  bc.PoolID,
			Address: bc.Address,
			Amount:  bc.Amount.InexactFloat64(),
			Usage:   bc.Usage,
			Created: bc.Created,
		}
	}
	return res
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
	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}
	addr := getMinerAddress(c, poolCfg)
	if addr == "" {
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}

	pageCount, err := s.db.GetMinerPaymentsByDayCount(c.Context(), poolCfg.ID, addr)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	page, pageSize := getPageParams(c)
	pageCount = uint(math.Floor(float64(pageCount) / float64(pageSize)))

	earnings, err := s.db.PageMinerPaymentsByDay(c.Context(), poolCfg.ID, addr, page, pageSize)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}

	res := DailyEarningRes{
		Meta: &Meta{
			Success:   true,
			PageCount: pageCount,
		},
		Result: dbEarningsToAPI(earnings),
	}
	return c.JSON(res)
}

func dbEarningsToAPI(earnings []*database.AmountByDate) []*DailyEarning {
	res := make([]*DailyEarning, len(earnings))
	for i, e := range earnings {
		res[i] = &DailyEarning{
			Amount: e.Amount.InexactFloat64(),
			Date:   e.Date,
		}
	}
	return res
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
// @Param perfMode query string Daily "Specify the sample range (default=day"
// @Success 200 {object} api.MinerPerformanceRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/performance [get]
func (s *Server) getMinerPerformanceHandler(c *fiber.Ctx) error {
	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}
	addr := getMinerAddress(c, poolCfg)
	if addr == "" {
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}
	mode := getPerformanceModeQuery(c)

	stats, err := s.getMinerPerformanceInternal(c.Context(), mode, poolCfg, addr)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	return c.JSON(&MinerPerformanceRes{
		Meta: &Meta{
			Success: true,
		},
		Result: dbPerformanceToAPIPerformance(stats),
	})
}

func dbPerformanceToAPIPerformance(stats []*database.PerformanceStats) []*PerformanceStats {
	res := make([]*PerformanceStats, len(stats))
	for i, s := range stats {
		res[i] = &PerformanceStats{
			Created:          s.Created,
			Hashrate:         s.Hashrate,
			ReportedHashrate: s.ReportedHashrate,
			SharesPerSecond:  s.SharesPerSecond,
		}
	}
	return res
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
