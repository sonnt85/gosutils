package vncproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

func TestProxy(t *testing.T) {
	r := gin.Default()

	vncProxy := NewVNCProxy()
	r.GET("/websockify", func(ctx *gin.Context) {
		h := websocket.Handler(vncProxy.ServeWS)
		h.ServeHTTP(ctx.Writer, ctx.Request)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/websockify")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
}

func NewVNCProxy() *Proxy {
	return New(&Config{
		Logger: logrus.StandardLogger(),
		TokenHandler: func(r *http.Request, retToken ...*string) (addr string, err error) {
			return "/tmp/.X11-unix/X0", nil
		},
		OnConnectFunc: func(*Peer) {
		},
		OnDisconectFunc: func(*Peer) {
		},
	})
}
