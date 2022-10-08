package api

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/phantomias/utils"
)

// @Summary Get overall stats
// @Description Get stats for all pools
// @Tags Overall
// @Produce  json
// @Success 200 {object} api.StatsRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/stats [get]
func (s *Server) getOverallPoolStatsHandler(c *fiber.Ctx) error {
	stats, err := s.db.GetOverallPoolStats(c.Context())
	if err != nil {
		return utils.SendAPIError(c, http.StatusInternalServerError, err)
	}
	res := &StatsRes{
		Meta: &Meta{
			Success: true,
		},
		Result: Stats(stats),
	}
	return c.JSON(res)
}

// @Summary Get overall stats
// @Description Get stats for all pools
// @Tags Overall
// @Produce  json
// @Param address query string true "Address to search for"
// @Success 200 {object} api.MinerSearchRes
// @Failure 400 {object} utils.APIError
// @Router /api/v1/search [get]
func (s *Server) getSearchMinerAddress(c *fiber.Ctx) error {
	addr := getAddressQuery(c)
	if addr == "" {
		return utils.SendAPIError(c, http.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}
	addresses, err := s.db.SearchMinerByAddress(c.Context(), addr)
	if err != nil {
		return utils.SendAPIError(c, http.StatusInternalServerError, err)
	}
	searchResults := make([]MinerSearch, 0, len(addresses))
	for _, a := range addresses {
		cfg := getPoolCfgByID(a.PoolID, s.pools)
		if cfg == nil {
			continue
		}
		searchResults = append(
			searchResults, MinerSearch{
				PoolID:  a.PoolID,
				FeeType: cfg.FeeType,
				Address: a.Address,
			},
		)
	}

	res := &MinerSearchRes{
		Meta:   &Meta{},
		Result: searchResults,
	}
	res.Meta.Success = len(searchResults) > 0
	return c.JSON(res)
}
