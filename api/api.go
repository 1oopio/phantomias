package api

import (
	"context"

	"github.com/1oopio/phantomias/config"
	"github.com/1oopio/phantomias/database"
	"github.com/1oopio/phantomias/price"
	"github.com/1oopio/phantomias/version"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/stratumfarm/go-miningcore-client"
)

// Server represents the api server of the proxy.
type Server struct {
	ctx              context.Context
	cancel           context.CancelFunc
	cfg              *config.Proxy
	mc               *miningcore.Client
	db               *database.DB
	api              *fiber.App
	wsRelay          *wsRelay
	price            price.Client
	metricsCollector fiber.Handler
	pools            []*config.Pool
}

// New creates a new server.
func New(ctx context.Context, cfg *config.Proxy, pools []*config.Pool, mc *miningcore.Client, db *database.DB, price price.Client, metricsCollector fiber.Handler) *Server {
	ctxc, cancel := context.WithCancel(ctx)
	s := &Server{
		ctx:    ctxc,
		cancel: cancel,
		api: fiber.New(fiber.Config{
			JSONEncoder:             json.Marshal,
			JSONDecoder:             json.Unmarshal,
			EnableTrustedProxyCheck: cfg.TrustedProxyCheck,
			TrustedProxies:          cfg.TrustedProxies,
			DisableStartupMessage:   version.Version != version.Development,
		}),
		mc:               mc,
		db:               db,
		cfg:              cfg,
		pools:            pools,
		wsRelay:          newWSRelay(ctx),
		price:            price,
		metricsCollector: metricsCollector,
	}

	s.api.Use(s.recover())
	if s.metricsCollector != nil {
		s.api.Use(s.metricsCollector)
	}

	s.setupRoutes()
	return s
}

// Start starts the api server.
// If a certFile and certKey is set, the server will use https.
func (s *Server) Start() error {
	go s.wsRelay.hub() // start the websocket relay

	if s.cfg.CertFile != "" && s.cfg.CertKey != "" {
		return s.api.ListenTLS(s.cfg.Listen, s.cfg.CertFile, s.cfg.CertKey)
	}
	return s.api.Listen(s.cfg.Listen)
}

// Close closes the server gracefully.
func (s *Server) Close() error {
	s.cancel()
	return s.api.Shutdown()
}

func (s *Server) BroadcastChan() chan []byte {
	return s.wsRelay.broadcast
}

func (s *Server) API() *fiber.App {
	return s.api
}
