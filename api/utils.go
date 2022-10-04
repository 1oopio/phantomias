package api

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/duration"
	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/phantomias/config"
	"github.com/stratumfarm/phantomias/database"
	"github.com/stratumfarm/phantomias/utils"
)

func handleAPIError(c *fiber.Ctx, code int, err error) error {
	if code == 0 {
		return utils.HandleMCError(c, err)
	}
	return utils.SendAPIError(c, code, err)
}

func getPoolCfgByID(id string, pools []*config.Pool) *config.Pool {
	for _, p := range pools {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func getPageParams(c *fiber.Ctx) (int, int) {
	page, err := strconv.Atoi(c.Query("page", "0"))
	if err != nil || page < 0 {
		page = 0
	}
	pageSize, err := strconv.Atoi(c.Query("pageSize", "15"))
	if err != nil || pageSize <= 0 {
		pageSize = 15
	}
	return page, pageSize
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

func getTopMinersRange(c *fiber.Ctx) int {
	topMinersRangeString := c.Query("topMinersRange", "1")
	topMinersRange, err := strconv.Atoi(topMinersRangeString)
	if err != nil {
		return 1
	}
	if topMinersRange < 1 || topMinersRange > 24 {
		topMinersRange = 1
	}
	return topMinersRange
}

func getEffortRange(c *fiber.Ctx) int {
	effortRangeString := c.Query("effortRange", "50")
	effortRange, err := strconv.Atoi(effortRangeString)
	if err != nil {
		return 50
	}
	if effortRange < 1 {
		effortRange = 50
	}
	return effortRange
}

func getPerformanceModeQuery(c *fiber.Ctx) database.SampleRange {
	switch c.Query("perfMode", "day") {
	case "hour":
		return database.RangeHour
	case "day":
		return database.RangeDay
	case "month":
		return database.RangeMonth
	default:
		return database.RangeDay
	}
}

func getMinerAddress(c *fiber.Ctx, poolCfg *config.Pool) string {
	addr := c.Params("miner_addr")
	if strings.EqualFold(poolCfg.Type, "ethereum") {
		addr = strings.ToLower(addr)
	}
	return addr
}

func getWorkerName(c *fiber.Ctx) string {
	return c.Params("worker_name")
}

func getPerformanceRange(mode database.SampleRange) (start, end time.Time) {
	end = time.Now()
	switch mode {
	case database.RangeHour:
		end = end.Add(-time.Second)
		start = end.Add(-time.Hour)
		return

	case database.RangeDay:
		if end.Minute() < 30 {
			end = end.Add(-time.Hour)
		}
		end = end.Add(-time.Minute)
		end = end.Add(-time.Second)
		start = end.Add(-time.Hour * 24)
		return

	case database.RangeMonth:
		end = end.Add(-time.Second)
		start = start.Add(-duration.Month)
		return

	default:
		if end.Minute() < 30 {
			end = end.Add(-time.Hour)
		}
		end = end.Add(-time.Minute)
		end = end.Add(-time.Second)
		start = end.Add(-time.Hour * 24)
		return
	}
}

func (s *Server) getMinerPerformanceInternal(ctx context.Context, mode database.SampleRange, poolCfg *config.Pool, addr string) ([]*PerformanceStats, error) {
	start, end := getPerformanceRange(mode)
	stats, err := s.db.GetMinerPerformanceBetweenTenMinutely(ctx, poolCfg.ID, addr, start, end)
	if err != nil {
		return nil, err
	}
	res := make([]*PerformanceStats, len(stats))
	for i, s := range stats {
		res[i] = &PerformanceStats{
			Created:          s.Created,
			Hashrate:         utils.ValueOrZero(s.Hashrate),
			ReportedHashrate: utils.ValueOrZero(s.ReportedHashrate),
			SharesPerSecond:  utils.ValueOrZero(s.SharesPerSecond),
			WorkersOnline:    s.WorkersOnline,
		}
	}
	return res, nil
}

func (s *Server) getWorkerPerformanceInternal(ctx context.Context, mode database.SampleRange, poolCfg *config.Pool, addr, worker string) ([]*PerformanceStats, error) {
	start, end := getPerformanceRange(mode)
	stats, err := s.db.GetWorkerPerformanceBetweenTenMinutely(ctx, poolCfg.ID, addr, worker, start, end)
	if err != nil {
		return nil, err
	}
	res := make([]*PerformanceStats, len(stats))
	for i, s := range stats {
		res[i] = &PerformanceStats{
			Created:          s.Created,
			Hashrate:         utils.ValueOrZero(s.Hashrate),
			ReportedHashrate: utils.ValueOrZero(s.ReportedHashrate),
			SharesPerSecond:  utils.ValueOrZero(s.SharesPerSecond),
			WorkersOnline:    s.WorkersOnline,
		}
	}
	return res, nil
}
