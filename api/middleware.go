package api

import (
	"time"
	"unsafe"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func (s *Server) recover() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
	})
}

func (s *Server) ratelimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.IP() == "127.0.0.1"
		},
		Max:               1000,
		Expiration:        time.Second * 10,
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

func (s *Server) cache() fiber.Handler {
	cfg := cache.ConfigDefault
	cfg.Next = func(c *fiber.Ctx) bool {
		return c.Query("refresh") == "true"
	}
	cfg.Expiration = s.cfg.CacheTTL
	cfg.CacheControl = true
	cfg.MaxBytes = 1000000000
	cfg.KeyGenerator = func(c *fiber.Ctx) string {
		q := c.Context().QueryArgs().QueryString()
		return c.Path() + *(*string)(unsafe.Pointer(&q))
	}
	return cache.New(cfg)
}
