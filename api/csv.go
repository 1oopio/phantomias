package api

import (
	"errors"
	"os"
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

const tmpCSVDir = "./tmp_csv"

func init() {
	// make sure ./tmp directory exists
	if err := os.MkdirAll(tmpCSVDir, 0755); err != nil {
		panic(err)
	}
}

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

		f, err := os.CreateTemp(tmpCSVDir, "csv-download")
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}
		defer os.Remove(f.Name())
		defer f.Close()

		if err := gocsv.Marshal(payments, f); err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}
		return c.Download(f.Name(), "payouts.csv")

	case csvDataEarnings:
		earnings, err := s.db.GetMinerPaymentsByDayBetween(c.UserContext(), poolCfg.ID, addr, start, end)
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}

		f, err := os.CreateTemp(tmpCSVDir, "csv-download")
		if err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}
		defer os.Remove(f.Name())
		defer f.Close()

		if err := gocsv.Marshal(earnings, f); err != nil {
			return utils.SendAPIError(c, fiber.StatusInternalServerError, err)
		}
		return c.Download(f.Name(), "earnings.csv")
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
