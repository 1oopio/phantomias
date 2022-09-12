package api

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/phantomias/config"
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

func sprintfOrEmpty(s string, args ...any) string {
	if s != "" {
		return fmt.Sprintf(s, args...)
	}
	return ""
}
