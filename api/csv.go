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
	"github.com/stratumfarm/phantomias/utils"
)

type csvDataValue string

const (
	csvDataHashrate csvDataValue = "hashrate"
	csvDataPayouts  csvDataValue = "payouts"
	csvDataEarnings csvDataValue = "earnings"
)

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
	if end.Sub(start) > duration.Month {
		return utils.SendAPIError(c, fiber.StatusBadRequest, errors.New("time range cannot be greater than 1 month"))
	}

	switch dataQuery {
	case csvDataHashrate:
	case csvDataPayouts:
		payments, err := s.db.GetMinerPaymentsBetween(c.UserContext(), poolCfg.ID, addr, start, end)
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		buff := bytes.NewBuffer(nil)
		if err := gocsv.Marshal(payments, buff); err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		setCSVFileNameHeader(c, "payouts.csv")
		return c.SendStream(buff)

	case csvDataEarnings:
		earnings, err := s.db.GetMinerPaymentsByDayBetween(c.UserContext(), poolCfg.ID, addr, start, end)
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		buff := bytes.NewBuffer(nil)
		if err := gocsv.Marshal(earnings, buff); err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		setCSVFileNameHeader(c, "earnings.csv")
		return c.SendStream(buff)
	}

	return nil
}

func getCSVDataQuery(c *fiber.Ctx) csvDataValue {
	if data := c.Query("data"); data != "" {
		for _, v := range []csvDataValue{csvDataHashrate, csvDataPayouts, csvDataEarnings} {
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
