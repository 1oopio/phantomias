package recws

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jpillora/backoff"
)

const (
	closedState = 1 << iota
	connectingState
	connectedState
	closedForeverState
)

var (
	// ErrNotConnected is returned when the application read/writes
	// a message and the connection is closed
	ErrNotConnected = errors.New("websocket: not connected")
)

type RecConn interface {
	// Close closes the underlying network connection without
	// sending or waiting for a close frame.
	Close(forever bool)

	// CloseAndReconnect closes the underlying connection and tries to reconnect.
	CloseAndReconnect()

	// Shutdown gracefully closes the connection by sending the websocket.CloseMessage.
	// The writeWait param defines the duration before the deadline of the write operation is hit.
	Shutdown(writeWait time.Duration)

	// ReadMessage is a helper method for getting a reader
	// using NextReader and reading from that reader to a buffer.
	//
	// If the connection is closed ErrNotConnected is returned.
	ReadMessage() (messageType int, message []byte, err error)

	// WriteMessage is a helper method for getting a writer using NextWriter,
	// writing the message and closing the writer.
	//
	// If the connection is closed ErrNotConnected is returned.
	WriteMessage(messageType int, data []byte) error

	// WriteJSON writes the JSON encoding of v to the connection.
	//
	// See the documentation for encoding/json Marshal for details about the
	// conversion of Go values to JSON.
	//
	// If the connection is closed ErrNotConnected is returned.
	WriteJSON(v interface{}) error

	// ReadJSON reads the next JSON-encoded message from the connection and stores
	// it in the value pointed to by v.
	//
	// See the documentation for the encoding/json Unmarshal function for details
	// about the conversion of JSON to a Go value.
	//
	// If the connection is closed ErrNotConnected is returned.
	ReadJSON(v interface{}) error

	// Dial creates a new client connection by calling DialContext with a background context.
	Dial() error

	// DialContext creates a new client connection.
	DialContext(ctx context.Context) error

	// GetURL returns connection url.
	GetURL() string

	// GetHTTPResponse returns the http response from the handshake.
	// Useful when WebSocket handshake fails,
	// so that callers can handle redirects, authentication, etc.
	GetHTTPResponse() *http.Response

	// IsConnected returns true if the websocket client is connected to the server.
	IsConnected() bool
}

// The recConn type represents a Reconnecting WebSocket connection.
type recConn struct {
	opts            ConnOps
	state           int
	mu              sync.RWMutex
	url             string
	reqHeader       http.Header
	httpResp        *http.Response
	dialer          *websocket.Dialer
	keepaliveCtx    context.Context
	keepaliveCancel context.CancelFunc

	*websocket.Conn
}

// New creates new Reconnecting Websocket connection
// The `url` parameter specifies the host and request URI. Use `requestHeader` to specify
// the origin (Origin), subprotocols (Sec-WebSocket-Protocol) and cookies (Cookie).
// Use GetHTTPResponse() method for the response.Header to get
// the selected subprotocol (Sec-WebSocket-Protocol) and cookies (Set-Cookie).
func New(url string, requestHeader http.Header, opts ...Opts) (RecConn, error) {
	if valid, err := isValidUrl(url); !valid {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	const (
		defaultReconnectIntervalMin    = 2 * time.Second
		defaultReconnectIntervalMax    = 30 * time.Second
		defaultReconnectIntervalFactor = 1.5
		defaultHandshakeTimeout        = 2 * time.Second
		defaultKeepAliveTimeout        = 0
	)

	var (
		defaultProxy                            = http.ProxyFromEnvironment
		defaultOnConnectionCallback             = func() {}
		defaultTLSClientConfig      *tls.Config = nil
		defaultLogFnDebug                       = func(s string) {}
		defaultLogFnError                       = func(err error, s string) {}
	)

	rc := recConn{
		opts: ConnOps{
			ReconnectIntervalMin:    defaultReconnectIntervalMin,
			ReconnectIntervalMax:    defaultReconnectIntervalMax,
			ReconnectIntervalFactor: defaultReconnectIntervalFactor,
			HandshakeTimeout:        defaultHandshakeTimeout,
			Proxy:                   defaultProxy,
			TLSClientConfig:         defaultTLSClientConfig,
			OnConnectCallback:       defaultOnConnectionCallback,
			KeepAliveTimeout:        defaultKeepAliveTimeout,
			LogFn: logFnOptions{
				Debug: defaultLogFnDebug,
				Error: defaultLogFnError,
			},
		},
		url:       url,
		reqHeader: requestHeader,
		state:     closedState,
	}

	for _, opt := range opts {
		opt(&rc.opts)
	}

	rc.dialer = &websocket.Dialer{
		HandshakeTimeout: rc.opts.HandshakeTimeout,
		Proxy:            rc.opts.Proxy,
		TLSClientConfig:  rc.opts.TLSClientConfig,
	}

	return &rc, nil
}

func (rc *recConn) CloseAndReconnect() {
	rc.Close(false)
	go func() {
		if err := rc.reconnect(); err != nil {
			rc.opts.LogFn.Error(err, "connection error")
		}
	}()
}

func (rc *recConn) Close(forever bool) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.keepaliveCancel != nil {
		rc.keepaliveCancel()
	}

	if rc.state&(closedState|closedForeverState) > 0 {
		return
	}
	if err := rc.Conn.Close(); err != nil {
		rc.opts.LogFn.Error(err, "websocket connection closing error")
	}
	if forever {
		rc.state = closedForeverState
	} else {
		rc.state = closedState
	}
}

func (rc *recConn) Shutdown(writeWait time.Duration) {
	if rc.isState(closedState | closedForeverState) {
		return
	}
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	err := rc.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
	if err != nil && err != websocket.ErrCloseSent {
		rc.opts.LogFn.Error(err, "shutdown error")
		// If close message could not be sent, then close without the handshake.
		rc.Close(true)
	}
}

func (rc *recConn) ReadMessage() (messageType int, message []byte, err error) {
	if !rc.IsConnected() {
		return messageType, message, ErrNotConnected
	}

	messageType, message, err = rc.Conn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) && rc.opts.RespectServerClosure {
			rc.Close(true)
			return messageType, message, nil
		}
		rc.CloseAndReconnect()
	}

	return
}

func (rc *recConn) WriteMessage(messageType int, data []byte) (err error) {
	if !rc.IsConnected() {
		return ErrNotConnected
	}

	rc.mu.Lock()
	err = rc.Conn.WriteMessage(messageType, data)
	rc.mu.Unlock()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) && rc.opts.RespectServerClosure {
			rc.Close(true)
			return nil
		}
		rc.CloseAndReconnect()
	}

	return err
}

func (rc *recConn) WriteJSON(v interface{}) (err error) {
	if !rc.IsConnected() {
		return ErrNotConnected
	}

	rc.mu.Lock()
	err = rc.Conn.WriteJSON(v)
	rc.mu.Unlock()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) && rc.opts.RespectServerClosure {
			rc.Close(true)
			return nil
		}
		rc.CloseAndReconnect()
	}

	return err
}

func (rc *recConn) ReadJSON(v interface{}) (err error) {
	if !rc.IsConnected() {
		return ErrNotConnected
	}

	err = rc.Conn.ReadJSON(v)
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) && rc.opts.RespectServerClosure {
			rc.Close(true)
			return nil
		}
		rc.CloseAndReconnect()
	}

	return err
}

func (rc *recConn) Dial() error {
	return rc.DialContext(context.Background())
}

func (rc *recConn) DialContext(ctx context.Context) error {
	if !rc.setStateIfNot(connectingState, connectingState|connectedState) {
		return nil
	}

	if err := rc.dial(ctx); err != nil {
		return err
	}

	return nil
}

func (rc *recConn) dial(ctx context.Context) error {
	wsConn, httpResp, err := rc.dialer.DialContext(ctx, rc.url, rc.reqHeader)

	rc.mu.Lock()
	rc.Conn = wsConn
	if err == nil {
		rc.state = connectedState
	} else {
		rc.state = closedState
	}
	rc.httpResp = httpResp
	rc.mu.Unlock()

	if err != nil {
		return err
	}

	rc.opts.OnConnectCallback()
	if rc.IsKeepAliveEnabled() {
		rc.keepAlive()
	}
	rc.opts.LogFn.Debug(fmt.Sprintf("Dial: connection successfully established with %s", rc.url))

	return nil
}

func (rc *recConn) reconnect() error {
	if !rc.setStateIfNot(connectingState, connectingState|connectedState|closedForeverState) {
		return nil
	}

	b := rc.makeBackoff()
	rand.Seed(time.Now().UTC().UnixNano())

	for {
		err := rc.dial(context.Background())
		if err == nil {
			return nil
		}

		waitDuration := b.Duration()
		rc.opts.LogFn.Error(err, fmt.Sprintf("dial error, will try again in %f seconds", waitDuration.Seconds()))
		time.Sleep(waitDuration)
	}
}

func (rc *recConn) GetURL() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.url
}

func (rc *recConn) makeBackoff() backoff.Backoff {
	return backoff.Backoff{
		Min:    rc.opts.ReconnectIntervalMin,
		Max:    rc.opts.ReconnectIntervalMax,
		Factor: rc.opts.ReconnectIntervalFactor,
		Jitter: true,
	}
}

func (rc *recConn) writeControlPingMessage() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	return rc.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
}

func (rc *recConn) keepAlive() {
	var keepAliveResponse = new(keepAliveResponse)
	rc.mu.Lock()
	rc.Conn.SetPongHandler(func(msg string) error {
		keepAliveResponse.setLastResponse()
		return nil
	})
	if rc.keepaliveCancel != nil { // cancel previous context if there's any
		rc.keepaliveCancel()
	}
	rc.keepaliveCtx, rc.keepaliveCancel = context.WithCancel(context.Background())
	rc.mu.Unlock()

	go func() {
		var ticker = time.NewTicker(rc.opts.KeepAliveTimeout)
		defer ticker.Stop()

		if err := rc.writeControlPingMessage(); err != nil {
			rc.opts.LogFn.Error(err, "error in writing ping message")
		}

		for {
			select {
			case <-rc.keepaliveCtx.Done():
				return
			case <-ticker.C:
				if err := rc.writeControlPingMessage(); err != nil {
					rc.opts.LogFn.Error(err, "error in writing ping message")
				}
			}
		}
	}()

	go func() {
		var ticker = time.NewTicker(rc.opts.KeepAliveTimeout)
		defer ticker.Stop()

		for {
			select {
			case <-rc.keepaliveCtx.Done():
				return
			case tick := <-ticker.C:
				if tick.Sub(keepAliveResponse.getLastResponse().Add(time.Millisecond)) > rc.opts.KeepAliveTimeout {
					go rc.CloseAndReconnect()
					return
				}
			}
		}
	}()
}

func (rc *recConn) setState(state int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.state = state
}

func (rc *recConn) isState(s int) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return (rc.state & s) > 0
}

func (rc *recConn) setStateIfNot(targetState, conditionState int) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if rc.state&conditionState == 0 {
		rc.state = targetState
		return true
	}

	return false
}

func (rc *recConn) GetHTTPResponse() *http.Response {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.httpResp
}

func (rc *recConn) IsConnected() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.state == connectedState
}

func (rc *recConn) IsKeepAliveEnabled() bool {
	return rc.opts.KeepAliveTimeout > 0
}

// isValidUrl checks whether url is valid
func isValidUrl(urlStr string) (bool, error) {
	if urlStr == "" {
		return false, fmt.Errorf("url cannot be empty")
	}

	u, err := url.Parse(urlStr)

	if err != nil {
		return false, err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return false, fmt.Errorf("websocket uris must start with ws or wss scheme")
	}

	if u.User != nil {
		return false, fmt.Errorf("user name and password are not allowed in websocket URIs")
	}

	return true, nil
}
