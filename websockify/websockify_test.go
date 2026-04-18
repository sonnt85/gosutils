package websockify

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// echoServer starts a TCP server that echoes every byte back. Stops on stop().
func echoServer(t *testing.T) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					if _, err := c.Write(buf[:n]); err != nil {
						return
					}
				}
			}(c)
		}
	}()
	return ln.Addr().String(), func() {
		_ = ln.Close()
		wg.Wait()
	}
}

func newProxyServer(target string) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("/websockify", Handler(target))
	return httptest.NewServer(mux)
}

func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/websockify"
	d := websocket.Dialer{Subprotocols: []string{"binary"}}
	c, _, err := d.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return c
}

func TestProxyRoundTrip(t *testing.T) {
	addr, stop := echoServer(t)
	t.Cleanup(stop)

	srv := newProxyServer(addr)
	t.Cleanup(srv.Close)

	c := dialWS(t, srv)
	t.Cleanup(func() { _ = c.Close() })

	msg := []byte("hello-websockify")
	if err := c.WriteMessage(websocket.BinaryMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, got, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("echo mismatch: got %q want %q", got, msg)
	}
}

// TestProxyNoLeakOnClientClose reproduces the original bug: select{} in the
// former handleWss caused the handler plus both io.Copy goroutines to linger
// forever after the client disconnected. With the fix, every goroutine must
// unwind promptly when the client closes.
func TestProxyNoLeakOnClientClose(t *testing.T) {
	addr, stop := echoServer(t)
	t.Cleanup(stop)

	srv := newProxyServer(addr)
	t.Cleanup(srv.Close)

	// Warm-up session stabilizes runtime-level goroutines used by net/http.
	warm := dialWS(t, srv)
	if err := warm.WriteMessage(websocket.BinaryMessage, []byte("w")); err != nil {
		t.Fatalf("warm write: %v", err)
	}
	_ = warm.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := warm.ReadMessage(); err != nil {
		t.Fatalf("warm read: %v", err)
	}
	_ = warm.Close()
	time.Sleep(100 * time.Millisecond)

	baseline := runtime.NumGoroutine()

	const sessions = 20
	for i := 0; i < sessions; i++ {
		c := dialWS(t, srv)
		if err := c.WriteMessage(websocket.BinaryMessage, []byte("ping")); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		_ = c.SetReadDeadline(time.Now().Add(time.Second))
		if _, _, err := c.ReadMessage(); err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		_ = c.Close()
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine()-baseline < 5 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("goroutine leak after %d sessions: baseline=%d now=%d", sessions, baseline, runtime.NumGoroutine())
}

// TestProxyClosesOnBackendClose verifies the other direction of the fix: when
// the backend closes, the WebSocket side must also close so the remote client
// sees EOF instead of hanging forever.
func TestProxyClosesOnBackendClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// Backend: echo one message, then close.
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 128)
		n, err := c.Read(buf)
		if err != nil {
			return
		}
		_, _ = c.Write(buf[:n])
	}()

	srv := newProxyServer(ln.Addr().String())
	t.Cleanup(srv.Close)

	c := dialWS(t, srv)
	t.Cleanup(func() { _ = c.Close() })

	if err := c.WriteMessage(websocket.BinaryMessage, []byte("probe")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := c.ReadMessage(); err != nil {
		t.Fatalf("read echo: %v", err)
	}

	// After backend closes, subsequent read on WS must error (close/EOF),
	// not block past the deadline. A blocking read means the proxy kept the
	// WS side alive while the TCP side was gone.
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("expected close/EOF after backend closed, got no error")
	}
}
