package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

var wsHandshakeTimeout = time.Second * 20

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
	v1.Get("/stats", cache, s.getOverallPoolStatsHandler)
	v1.Get("/search", s.getSearchMinerAddress)
	// pools
	v1.Get("/pools", cache, s.getPoolsHandler)
	v1.Get("/pools/:id", s.getPoolHandler)
	v1.Get("pools/:id/blocks", s.getBlocksHandler)
	v1.Get("pools/:id/payments", s.getPaymentsHandler)
	v1.Get("pools/:id/performance", cache, s.getPoolPerformanceHandler)

	// miners
	v1.Get("pools/:id/miners", s.getMinersHandler)
	v1.Get("pools/:id/miners/:miner_addr", s.getMinerHandler)
	v1.Get("pools/:id/miners/:miner_addr/payments", s.getMinerPaymentsHandler)
	v1.Get("pools/:id/miners/:miner_addr/balancechanges", s.getMinerBalanceChangesHandler)
	v1.Get("pools/:id/miners/:miner_addr/earnings/daily", s.getMinerDailyEarningsHandler)
	v1.Get("pools/:id/miners/:miner_addr/performance", cache, s.getMinerPerformanceHandler)
	v1.Get("pools/:id/miners/:miner_addr/settings", s.getMinerSettingsHandler)
	v1.Post("pools/:id/miners/:miner_addr/settings", s.postMinerSettingsHandler)

	// workers
	v1.Get("pools/:id/miners/:miner_addr/workers/:worker_name/performance", cache, s.getWorkerHandler)
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
	s.api.Get("/swagger/*", append(middleware, s.swagger())...)
}

// @Summary Teapot
// @Tags Teapot
// @Produce text/plain
// @Success 418 {string} string "I'm a teapot"
// @Router /teapot [get]
func (s *Server) teaPot(middleware ...fiber.Handler) {
	s.api.Get("/teapot", append(middleware, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusTeapot)
	})...)
}
