package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/go-miningcore-client"
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
	poolInfo, code, err := s.mc.GetPools(c.Context())
	if err != nil {
		return handleAPIError(c, code, err)
	}
	result := poolsInfosToAPIPools(poolInfo)

	for _, p := range result {
		prices := s.price.GetPrices(strings.ToLower(p.Name))
		if prices != nil {
			priceRes := make(map[string]Price)
			for _, p := range prices {
				priceRes[p.VSCurrency] = Price{p.Price, p.PriceChangePercentage24H}
			}
			p.Prices = priceRes
		}
	}

	return c.Status(code).JSON(&PoolsRes{
		Meta: &Meta{
			Success: true,
		},
		Result: result,
	})
}

func handleAPIError(c *fiber.Ctx, code int, err error) error {
	if code == 0 {
		return utils.HandleMCError(c, err)
	}
	return utils.SendAPIError(c, code, err)
}

func poolsInfosToAPIPools(info []*miningcore.PoolInfo) []*Pool {
	pools := make([]*Pool, len(info))
	for i, p := range info {
		pools[i] = poolInfoToAPIPool(p)
	}
	return pools
}

func poolInfoToAPIPool(p *miningcore.PoolInfo) *Pool {
	return &Pool{
		Coin:            p.Coin.Symbol,
		ID:              p.ID,
		Algorithm:       p.Coin.Algorithm,
		Name:            p.Coin.Name,
		Hashrate:        p.PoolStats.PoolHashrate,
		Miners:          p.PoolStats.ConnectedMiners,
		Fee:             p.PoolFeePercent,
		FeeType:         p.PaymentProcessing.PayoutScheme,
		BlockHeight:     p.NetworkStats.BlockHeight,
		NetworkHashrate: p.NetworkStats.NetworkHashrate,
	}
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
	poolInfo, code, err := s.mc.GetPool(c.Context(), c.Params("id"))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	res := &PoolExtendedRes{
		Meta: &Meta{
			Success: true,
		},
		Result: poolInfoToAPIPoolExtended(poolInfo),
	}

	prices := s.price.GetPrices(strings.ToLower(res.Result.Name))
	if prices != nil {
		priceRes := make(map[string]Price)
		for _, p := range prices {
			priceRes[p.VSCurrency] = Price{p.Price, p.PriceChangePercentage24H}
		}
		res.Result.Prices = priceRes
	}
	return c.Status(code).JSON(res)
}

func poolInfoToAPIPoolExtended(p *miningcore.PoolInfo) *PoolExtended {
	lastBlockFound, _ := strconv.ParseInt(p.LastPoolBlockTime, 10, 64)
	return &PoolExtended{
		Pool:               poolInfoToAPIPool(p),
		TotalBlocksFound:   p.TotalBlocks,
		TotalPayments:      p.TotalPaid,
		LastBlockFoundTime: lastBlockFound,
		Ports:              poolPortsToAPIPoolPorts(p.Ports),
	}
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
	var blocks BlocksRes
	code, err := s.mc.UnmarshalPoolBlocks(c.Context(), c.Params("id"), &blocks, handlePaginationQueries(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(blocks)
}

func handlePaginationQueries(c *fiber.Ctx) map[string]string {
	page := c.Query("page")
	pageSize := c.Query("pageSize")

	if page == "" && pageSize == "" {
		return nil
	}

	params := make(map[string]string)
	if page != "" {
		params["page"] = page
	}
	if pageSize != "" {
		params["pageSize"] = pageSize
	}
	return params
}

/* type Blockstatus int

const (
	BlockStatusUnknown Blockstatus = iota
	BlockStatusPending
	BlockStatusOrphaned
	BlockStatusConfirmed
) */

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
	var payments PaymentsRes
	code, err := s.mc.UnmarshalPoolPayments(c.Context(), c.Params("id"), &payments, handlePaginationQueries(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(payments)
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
	var performance struct {
		Stats []*PoolPerformance `json:"stats"`
	}
	code, err := s.mc.UnmarshalPoolPerformance(c.Context(), c.Params("id"), &performance, handlePerformanceQueries(c))
	if err != nil {
		return handleAPIError(c, code, err)
	}
	return c.Status(code).JSON(&PoolPerformanceRes{
		Meta: &Meta{
			Success: true,
		},
		Result: performance.Stats,
	})
}

func handlePerformanceQueries(c *fiber.Ctx) map[string]string {
	i := c.Query("i")
	r := c.Query("r")

	if i == "" && r == "" {
		return nil
	}

	params := make(map[string]string)
	if i != "" {
		params["i"] = i
	}
	if r != "" {
		params["r"] = r
	}
	return params
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
