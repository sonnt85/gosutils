package websockify

import (
	"flag"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// defaultUpgrader preserves the permissive origin policy of the previous
// x/net/websocket implementation. Callers exposing this on public endpoints
// should gate upstream (reverse proxy, auth middleware, or a bespoke Upgrader).
var defaultUpgrader = websocket.Upgrader{
	Subprotocols: []string{"binary"},
	CheckOrigin:  func(r *http.Request) bool { return true },
}

// Handler returns an http.Handler that upgrades requests to WebSocket and
// proxies bytes to/from the given TCP target. It can be mounted at any path
// on any http.ServeMux.
func Handler(targetAddr string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsconn, err := defaultUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("websockify upgrade:", err)
			return
		}
		handleProxy(wsconn, targetAddr)
	})
}

// New starts a standalone WebSocket-to-TCP proxy on sourceAddr serving the
// path "/websockify". It blocks until the HTTP server returns an error.
// Prefer Handler() when embedding into an existing HTTP server.
func New(sourceAddr, targetAddr string) error {
	mux := http.NewServeMux()
	mux.Handle("/websockify", Handler(targetAddr))
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

func handleProxy(wsconn *websocket.Conn, target string) {
	defer wsconn.Close()

	conn, err := net.Dial("tcp", target)
	if err != nil {
		log.Println("websockify dial backend:", err)
		return
	}
	defer conn.Close()

	// When either direction finishes, both deferred Close() unblock the paired reader.
	done := make(chan struct{}, 2)

	// TCP → WS: read backend bytes, wrap as binary WS messages.
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, rerr := conn.Read(buf)
			if n > 0 {
				if werr := wsconn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if rerr != nil {
				return
			}
		}
	}()

	// WS → TCP: stream each WS message payload into the backend socket.
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			_, r, rerr := wsconn.NextReader()
			if rerr != nil {
				return
			}
			if _, werr := io.Copy(conn, r); werr != nil {
				return
			}
		}
	}()

	<-done
}
