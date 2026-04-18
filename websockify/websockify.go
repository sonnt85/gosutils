package websockify

import (
	"context"
	"errors"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// defaultWriteBufferPool is shared by the default upgrader. It cuts alloc
// pressure when many concurrent sessions are active.
var defaultWriteBufferPool = &sync.Pool{}

// defaultUpgrader preserves the permissive origin policy of the previous
// x/net/websocket implementation. Callers exposing this on public endpoints
// should gate upstream (reverse proxy, auth middleware, or WithUpgrader).
var defaultUpgrader = websocket.Upgrader{
	Subprotocols:    []string{"binary"},
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	WriteBufferPool: defaultWriteBufferPool,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type options struct {
	upgrader     *websocket.Upgrader
	noDelay      bool
	dialTimeout  time.Duration
	pingInterval time.Duration
	writeTimeout time.Duration
	readBufSize  int
	onSession    func(rx, tx int64, dur time.Duration)
	onError      func(kind string, err error)
}

// Defaults are chosen so that Handler(target) without any options produces
// the same runtime behavior as before options existed: no extra syscalls,
// no mutex, no ping goroutine, no callbacks.
func newOptions(opts []Option) *options {
	o := &options{
		readBufSize: 32 * 1024,
	}
	for _, f := range opts {
		f(o)
	}
	return o
}

// Option tunes the behavior of Handler and New.
type Option func(*options)

// WithUpgrader replaces the default websocket.Upgrader. Use this to enable
// per-message-deflate compression, tighten CheckOrigin, or inject custom
// buffer sizing.
func WithUpgrader(u *websocket.Upgrader) Option {
	return func(o *options) { o.upgrader = u }
}

// WithNoDelay sets TCP_NODELAY on the backend connection. Default is false
// (Nagle enabled by OS). Set true for interactive traffic (SSH/terminal)
// where 40ms keystroke latency from Nagle is noticeable. Leave false for
// VNC or throughput-oriented traffic where small frames should coalesce.
func WithNoDelay(b bool) Option { return func(o *options) { o.noDelay = b } }

// WithDialTimeout caps the backend dial. Zero (default) uses the OS default,
// matching plain net.Dial behavior.
func WithDialTimeout(d time.Duration) Option { return func(o *options) { o.dialTimeout = d } }

// WithPingInterval sends a WebSocket ping every d; the session closes if the
// peer does not reply within 2*d. Zero (default) disables ping/pong and
// relies on TCP keepalive alone (Linux default: 2h).
func WithPingInterval(d time.Duration) Option { return func(o *options) { o.pingInterval = d } }

// WithWriteTimeout caps each individual WebSocket write. Zero (default) disables.
// Prevents a slow WS peer from pinning a goroutine indefinitely.
func WithWriteTimeout(d time.Duration) Option { return func(o *options) { o.writeTimeout = d } }

// WithReadBufferSize sets the TCP read buffer used when copying backend→WS.
// Default 32KB.
func WithReadBufferSize(n int) Option { return func(o *options) { o.readBufSize = n } }

// WithOnSession registers a callback fired when a proxy session ends. rx is
// bytes forwarded from WebSocket client to TCP backend; tx is the reverse.
func WithOnSession(f func(rx, tx int64, duration time.Duration)) Option {
	return func(o *options) { o.onSession = f }
}

// WithOnError registers a callback for non-fatal session errors. kind
// identifies the phase: "upgrade", "dial", "read", "write", "ping".
func WithOnError(f func(kind string, err error)) Option {
	return func(o *options) { o.onError = f }
}

// Handler returns an http.Handler that upgrades requests to WebSocket and
// proxies bytes to/from the given TCP target. It can be mounted at any path
// on any http.ServeMux. Pass Options to enable features like NoDelay,
// Ping/Pong, timeouts, and metrics callbacks.
func Handler(targetAddr string, opts ...Option) http.Handler {
	o := newOptions(opts)
	up := o.upgrader
	if up == nil {
		up = &defaultUpgrader
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsconn, err := up.Upgrade(w, r, nil)
		if err != nil {
			reportError(o, "upgrade", err)
			return
		}
		handleProxy(r.Context(), wsconn, targetAddr, o)
	})
}

// New starts a standalone WebSocket-to-TCP proxy on sourceAddr serving the
// path "/websockify". It blocks until the HTTP server returns an error.
// Prefer Handler() when embedding into an existing HTTP server.
func New(sourceAddr, targetAddr string, opts ...Option) error {
	mux := http.NewServeMux()
	mux.Handle("/websockify", Handler(targetAddr, opts...))
	return http.ListenAndServe(sourceAddr, mux)
}

// NewFromFlags is a drop-in replacement for callers that used the previous
// flag-driven New(). It parses --source and --target from os.Args via a local
// FlagSet (no global flag pollution), then starts the proxy.
//
// MIGRATION NOTE:
// If you previously called websockify.New() as a CLI entry point that relied
// on the package-level --source / --target flags, rename your call to
// websockify.NewFromFlags(). For programmatic use, switch to
// websockify.New(sourceAddr, targetAddr) instead.
func NewFromFlags() error {
	fs := flag.NewFlagSet("websockify", flag.ExitOnError)
	source := fs.String("source", "", "source address")
	target := fs.String("target", "", "target address")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}
	if *source == "" || *target == "" {
		fs.PrintDefaults()
		os.Exit(1)
	}
	return New(*source, *target)
}

func reportError(o *options, kind string, err error) {
	if o.onError != nil {
		o.onError(kind, err)
		return
	}
	log.Printf("websockify %s: %v", kind, err)
}

func handleProxy(ctx context.Context, wsconn *websocket.Conn, target string, o *options) {
	defer wsconn.Close()
	start := time.Now()

	// Use plain net.Dial when no timeout is configured so the default path
	// stays identical to the pre-options code.
	var conn net.Conn
	var err error
	if o.dialTimeout > 0 {
		d := net.Dialer{Timeout: o.dialTimeout}
		conn, err = d.DialContext(ctx, "tcp", target)
	} else {
		conn, err = net.Dial("tcp", target)
	}
	if err != nil {
		reportError(o, "dial", err)
		return
	}
	defer conn.Close()

	if o.noDelay {
		if tcp, ok := conn.(*net.TCPConn); ok {
			_ = tcp.SetNoDelay(true)
		}
	}

	// Pong handler: reset read deadline when peer answers our ping.
	if o.pingInterval > 0 {
		_ = wsconn.SetReadDeadline(time.Now().Add(2 * o.pingInterval))
		wsconn.SetPongHandler(func(string) error {
			return wsconn.SetReadDeadline(time.Now().Add(2 * o.pingInterval))
		})
	}

	var rx, tx int64
	// wmu is only needed when a ping goroutine also writes. Fast path skips it.
	var wmu *sync.Mutex
	if o.pingInterval > 0 {
		wmu = &sync.Mutex{}
	}
	done := make(chan struct{}, 2)

	// Backend TCP → WS: read backend bytes, wrap as binary messages.
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, o.readBufSize)
		for {
			n, rerr := conn.Read(buf)
			if n > 0 {
				if wmu != nil {
					wmu.Lock()
				}
				if o.writeTimeout > 0 {
					_ = wsconn.SetWriteDeadline(time.Now().Add(o.writeTimeout))
				}
				werr := wsconn.WriteMessage(websocket.BinaryMessage, buf[:n])
				if wmu != nil {
					wmu.Unlock()
				}
				if werr != nil {
					reportError(o, "write", werr)
					return
				}
				tx += int64(n)
			}
			if rerr != nil {
				if !isClosedErr(rerr) {
					reportError(o, "read", rerr)
				}
				return
			}
		}
	}()

	// WS → backend TCP.
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, r, rerr := wsconn.NextReader()
			if rerr != nil {
				if !isClosedErr(rerr) {
					reportError(o, "read", rerr)
				}
				return
			}
			n, werr := io.Copy(conn, r)
			rx += n
			if werr != nil {
				reportError(o, "write", werr)
				return
			}
		}
	}()

	var cancelPing context.CancelFunc
	if o.pingInterval > 0 {
		var pingCtx context.Context
		pingCtx, cancelPing = context.WithCancel(ctx)
		go pingLoop(pingCtx, wsconn, wmu, o)
	}

	<-done
	if cancelPing != nil {
		cancelPing()
	}
	if o.onSession != nil {
		o.onSession(rx, tx, time.Since(start))
	}
}

func pingLoop(ctx context.Context, wsconn *websocket.Conn, wmu *sync.Mutex, o *options) {
	t := time.NewTicker(o.pingInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			wmu.Lock()
			if o.writeTimeout > 0 {
				_ = wsconn.SetWriteDeadline(time.Now().Add(o.writeTimeout))
			}
			err := wsconn.WriteMessage(websocket.PingMessage, nil)
			wmu.Unlock()
			if err != nil {
				reportError(o, "ping", err)
				return
			}
		}
	}
}

func isClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
		websocket.CloseAbnormalClosure,
	)
}
