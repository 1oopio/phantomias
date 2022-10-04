package api

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/phantomias/utils"
)

// @Summary Get a worker
// @Description Get a specific worker from a specific miner from a specific pool
// @Tags Workers
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param worker_name path string true "Name of the worker"
// @Param perfMode query string Daily "Specify the sample range (default=day"
// @Success 200 {object} api.WorkerPerformanceRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/workers/{worker_name}/performance [get]
func (s *Server) getWorkerHandler(c *fiber.Ctx) error {
	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, http.StatusNotFound, utils.ErrPoolNotFound)
	}
	addr := getMinerAddress(c, poolCfg)
	if addr == "" {
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}
	worker := getWorkerName(c)
	if worker == "" {
		return handleAPIError(c, http.StatusBadRequest, utils.ErrInvalidWorkerName)
	}
	mode := getPerformanceModeQuery(c)

	stats, err := s.getWorkerPerformanceInternal(c.Context(), mode, poolCfg, addr, worker)
	if err != nil {
		return handleAPIError(c, http.StatusInternalServerError, err)
	}
	return c.JSON(&WorkerPerformanceRes{
		Meta: &Meta{
			Success: true,
		},
		Result: stats,
	})
}
