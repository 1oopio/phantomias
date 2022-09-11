package recws

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

type Opts func(o *ConnOps)

func WithReconnectIntervalMin(interval time.Duration) Opts {
	return func(o *ConnOps) {
		o.ReconnectIntervalMin = interval
	}
}

func WithReconnectIntervalMax(interval time.Duration) Opts {
	return func(o *ConnOps) {
		o.ReconnectIntervalMax = interval
	}
}

func WithReconnectIntervalFactor(factor float64) Opts {
	return func(o *ConnOps) {
		o.ReconnectIntervalFactor = factor
	}
}

func WithHandshakeTimeout(to time.Duration) Opts {
	return func(o *ConnOps) {
		o.HandshakeTimeout = to
	}
}

func WithProxy(p func(*http.Request) (*url.URL, error)) Opts {
	return func(o *ConnOps) {
		o.Proxy = p
	}
}

func WithTLSClientConfig(c *tls.Config) Opts {
	return func(o *ConnOps) {
		o.TLSClientConfig = c
	}
}

func WithOnConnectCallback(cb func()) Opts {
	return func(o *ConnOps) {
		o.OnConnectCallback = cb
	}
}

func WithKeepAliveTimeout(kat time.Duration) Opts {
	return func(o *ConnOps) {
		o.KeepAliveTimeout = kat
	}
}

func WithDebugLogFn(fn func(string)) Opts {
	return func(o *ConnOps) {
		o.LogFn.Debug = fn
	}
}

func WithErrorLogFn(fn func(error, string)) Opts {
	return func(o *ConnOps) {
		o.LogFn.Error = fn
	}
}

func WithRespectServerClosure(r bool) Opts {
	return func(o *ConnOps) {
		o.RespectServerClosure = r
	}
}

type ConnOps struct {
	// ReconnectIntervalMin specifies the initial reconnecting interval,
	// default to 2 seconds.
	ReconnectIntervalMin time.Duration
	// ReconnectIntervalMax specifies the maximum reconnecting interval,
	// default to 30 seconds.
	ReconnectIntervalMax time.Duration
	// ReconnectIntervalFactor specifies the rate of increase of the reconnection
	// interval, default to 1.5.
	ReconnectIntervalFactor float64
	// RespectServerClosure specifies whether the client should stop trying to reconnect
	// if the server asked so by sending the normal closure message.
	RespectServerClosure bool
	// HandshakeTimeout specifies the duration for the handshake to complete,
	// default to 2 seconds.
	HandshakeTimeout time.Duration
	// Proxy specifies the proxy function for the dialer
	// defaults to ProxyFromEnvironment.
	Proxy func(*http.Request) (*url.URL, error)
	// Client TLS config to use on reconnect.
	TLSClientConfig *tls.Config
	// OnConnectCallback fires after the connection successfully establish.
	OnConnectCallback func()
	// KeepAliveTimeout is an interval for sending ping/pong messages
	// disabled if 0.
	KeepAliveTimeout time.Duration
	// LogFn is a set of functions to run for different logging levels.
	LogFn logFnOptions
}

type logFnOptions struct {
	Debug func(string)
	Error func(error, string)
}
