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
