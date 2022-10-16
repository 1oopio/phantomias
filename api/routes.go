package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/timeout"
	"github.com/gofiber/websocket/v2"
)

var (
	wsHandshakeTimeout = time.Second * 20
	shortTimeout       = time.Second * 30
	longTimeout        = time.Minute * 3
	extremeTimeout     = time.Minute * 10
)

func (s *Server) setupRoutes() {
	cache := s.cache()
	ratelimiter := s.ratelimiter()

	// pool api
	s.apiRoutes(cache, ratelimiter)
	// websockets
	s.wsRoute(ratelimiter)
	// swagger
	s.swaggerRoute(ratelimiter)
	// teapot :D
	s.teaPot(ratelimiter)
}

func (s *Server) apiRoutes(cache, ratelimiter fiber.Handler) {
	api := s.api.Group("/api")
	v1 := api.Group("/v1", ratelimiter)

	// overall
	v1.Get("/stats",
		cache,
		timeout.New(s.getOverallPoolStatsHandler, shortTimeout),
	)
	v1.Get("/search",
		timeout.New(s.getSearchMinerAddress, shortTimeout),
	)

	// pools
	v1.Get("/pools",
		cache,
		timeout.New(s.getPoolsHandler, shortTimeout),
	)
	v1.Get("/pools/:id",
		timeout.New(s.getPoolHandler, shortTimeout),
	)
	v1.Get("pools/:id/blocks",
		timeout.New(s.getBlocksHandler, shortTimeout),
	)
	v1.Get("pools/:id/payments",
		timeout.New(s.getPaymentsHandler, shortTimeout),
	)
	v1.Get("pools/:id/performance",
		cache,
		timeout.New(s.getPoolPerformanceHandler, longTimeout),
	)
	v1.Get("pools/:id/topminers",
		cache,
		timeout.New(s.getTopMinersHandler, shortTimeout),
	)

	// miners
	v1.Get("pools/:id/miners",
		timeout.New(s.getMinersHandler, shortTimeout),
	)
	v1.Get("pools/:id/miners/:miner_addr",
		timeout.New(s.getMinerHandler, longTimeout),
	)
	v1.Get("pools/:id/miners/:miner_addr/payments",
		timeout.New(s.getMinerPaymentsHandler, shortTimeout),
	)
	v1.Get("pools/:id/miners/:miner_addr/balancechanges",
		timeout.New(s.getMinerBalanceChangesHandler, shortTimeout),
	)
	v1.Get("pools/:id/miners/:miner_addr/earnings/daily",
		timeout.New(s.getMinerDailyEarningsHandler, shortTimeout),
	)
	v1.Get("pools/:id/miners/:miner_addr/performance",
		cache,
		timeout.New(s.getMinerPerformanceHandler, longTimeout),
	)
	v1.Get("pools/:id/miners/:miner_addr/settings",
		timeout.New(s.getMinerSettingsHandler, shortTimeout),
	)
	v1.Post("pools/:id/miners/:miner_addr/settings",
		timeout.New(s.postMinerSettingsHandler, shortTimeout),
	)

	// workers
	v1.Get("pools/:id/miners/:miner_addr/workers/:worker_name/performance",
		cache,
		timeout.New(s.getWorkerPerformanceHandler, shortTimeout),
	)
	v1.Get("pools/:id/miners/:miner_addr/workers/:worker_name",
		timeout.New(s.getWorkerHandler, longTimeout),
	)

	// CSV
	v1.Get("pools/:id/miners/:miner_addr/csv",
		// use a stricter rate limit for CSV downloads
		limiter.New(limiter.Config{
			Next: func(c *fiber.Ctx) bool {
				return c.IP() == "127.0.0.1"
			},
			Max:               30,
			Expiration:        time.Second * 60,
			LimiterMiddleware: limiter.FixedWindow{},
		}),
		timeout.New(s.getCSVDownloadHandler, extremeTimeout),
	)
}

func (s *Server) wsRoute(middleware ...fiber.Handler) {
	// require a connection upgrade to websocket
	s.api.Use("/v1/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	s.api.Get("/v1/ws", append(middleware, websocket.New(s.wsHandler, websocket.Config{
		HandshakeTimeout: wsHandshakeTimeout,
	}))...)
}

func (s *Server) swaggerRoute(middleware ...fiber.Handler) {
	s.api.Get("/swagger/*", append(middleware, timeout.New(s.swagger(), shortTimeout))...)
}

// @Summary Teapot
// @Tags Teapot
// @Produce text/plain
// @Success 418 {string} string "I'm a teapot"
// @Router /teapot [get]
func (s *Server) teaPot(middleware ...fiber.Handler) {
	s.api.Get("/teapot", append(middleware, timeout.New(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusTeapot)
	}, time.Second*10))...)
}
