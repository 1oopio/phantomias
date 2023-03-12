package metrics

import (
	"context"
	"log"

	"github.com/1oopio/phantomias/config"
	fiberprom "github.com/ansrivas/fiberprometheus"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Opt func(*Server)

func WithContext(ctx context.Context) Opt {
	return func(s *Server) {
		s.parentCtx = ctx
	}
}

func WithRegistry(r *prometheus.Registry) Opt {
	return func(s *Server) {
		s.registry = r
	}
}

type Server struct {
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc
	cfg       *config.Metrics
	server    *fiber.App
	registry  *prometheus.Registry
	fiber     fiber.Handler
}

func New(cfg *config.Metrics, opts ...Opt) *Server {
	s := &Server{
		registry: prometheus.NewRegistry(),
		server: fiber.New(fiber.Config{
			DisableStartupMessage: true,
		}),
	}
	s.cfg = cfg
	s.parentCtx = context.Background()
	for _, opt := range opts {
		opt(s)
	}
	s.ctx, s.cancel = context.WithCancel(s.parentCtx)

	s.fiber = fiberprom.New("",
		fiberprom.WithRegistry(s.registry),
		fiberprom.WithNamespace("miningcore_api_proxy"),
	).Middleware

	s.server.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	if s.cfg.User != "" && s.cfg.Password != "" {
		s.server.Use(basicauth.New(basicauth.Config{
			Users: map[string]string{
				s.cfg.User: s.cfg.Password,
			},
		}))
	}

	s.server.Get(s.cfg.Endpoint, adaptor.HTTPHandler(promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})))

	return s
}

func (s *Server) Start() error {
	log.Printf("[metrics][server] starting on %s", s.cfg.Listen)
	return s.server.Listen(s.cfg.Listen)
}

func (s *Server) Close() error {
	s.cancel()
	return s.server.Shutdown()
}

func (s *Server) Fiber() fiber.Handler {
	return s.fiber
}
