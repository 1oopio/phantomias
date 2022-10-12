package api

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/caarlos0/duration"
	"github.com/gocarina/gocsv"
	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/phantomias/database"
	"github.com/stratumfarm/phantomias/utils"
)

type csvDataValue string

const (
	csvDataStats    csvDataValue = "stats"
	csvDataPayouts  csvDataValue = "payouts"
	csvDataEarnings csvDataValue = "earnings"
)

var maxCSVDataAge = duration.Month

// @Summary Download data as CSV
// @Description Download miner specific data as CSV
// @Tags CSV
// @Produce json
// @Param pool_id path string true "ID of the pool"
// @Param miner_addr path string true "Address of the miner"
// @Param data query string true "Specify the data type (stats, payouts, earnings)"
// @Param start query string true "Start time (RFC3339 format)"
// @Param end query string true "End time (RFC3339 format)"
// @Success 200
// @Failure 400 {object} utils.APIError
// @Router /api/v1/pools/{pool_id}/miners/{miner_addr}/csv [get]
func (s *Server) getCSVDownloadHandler(c *fiber.Ctx) error {
	poolCfg := getPoolCfgByID(c.Params("id"), s.pools)
	if poolCfg == nil {
		return handleAPIError(c, fiber.StatusNotFound, utils.ErrPoolNotFound)
	}
	addr := getMinerAddressParam(c, poolCfg)
	if addr == "" {
		return handleAPIError(c, fiber.StatusBadRequest, utils.ErrInvalidMinerAddress)
	}
	dataQuery := getCSVDataQuery(c)
	if dataQuery == "" {
		return utils.SendAPIError(c, fiber.StatusBadRequest, errors.New("invalid data query"))
	}
	start, end, err := getCSVStartEndTime(c)
	if err != nil {
		return utils.SendAPIError(c, fiber.StatusBadRequest, errors.New("invalid start or end time"))
	}
	if start.After(end) {
		return utils.SendAPIError(c, fiber.StatusBadRequest, errors.New("start time cannot be after end time"))
	}
	if end.Sub(start) > maxCSVDataAge {
		return utils.SendAPIError(c, fiber.StatusBadRequest, fmt.Errorf("time range cannot be greater than %s", maxCSVDataAge))
	}

	switch dataQuery {
	case csvDataStats:
		var statsEntity []*database.PerformanceStatsEntity
		var err error

		if end.Sub(start) > duration.Week {
			statsEntity, err = s.db.GetMinerPerformanceBetweenDaily(c.UserContext(), poolCfg.ID, addr, start, end)
		} else {
			statsEntity, err = s.db.GetMinerPerformanceBetweenTenMinutely(c.UserContext(), poolCfg.ID, addr, start, end)
		}
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}
		stats := dbPerformanceToAPIPerformance(statsEntity)

		var buf bytes.Buffer
		if err := gocsv.Marshal(stats, &buf); err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		setCSVFileNameHeader(c, "stats.csv")
		return c.SendStream(&buf)

	case csvDataPayouts:
		payments, err := s.db.GetMinerPaymentsBetween(c.UserContext(), poolCfg.ID, addr, start, end)
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		var buf bytes.Buffer
		if err := gocsv.Marshal(payments, &buf); err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		setCSVFileNameHeader(c, "payouts.csv")
		return c.SendStream(&buf)

	case csvDataEarnings:
		earnings, err := s.db.GetMinerPaymentsByDayBetween(c.UserContext(), poolCfg.ID, addr, start, end)
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		var buf bytes.Buffer
		if err := gocsv.Marshal(earnings, &buf); err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		setCSVFileNameHeader(c, "earnings.csv")
		return c.SendStream(&buf)
	}
	return nil
}

func getCSVDataQuery(c *fiber.Ctx) csvDataValue {
	if data := c.Query("data"); data != "" {
		for _, v := range []csvDataValue{csvDataStats, csvDataPayouts, csvDataEarnings} {
			if csvDataValue(data) == v {
				return csvDataValue(data)
			}
		}
	}
	return ""
}

func getCSVStartEndTime(c *fiber.Ctx) (time.Time, time.Time, error) {
	start := c.Query("start")
	end := c.Query("end")
	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return startTime, endTime, nil
}

func setCSVFileNameHeader(c *fiber.Ctx, name string) {
	c.Response().Header.Set(fiber.HeaderContentType, "text/csv")
	c.Response().Header.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", url.QueryEscape(name)))
}
