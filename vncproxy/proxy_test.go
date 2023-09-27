package vncproxy

import (
	"net/http"
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

	if err := r.Run(); err != nil {
		panic(err)
	}
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
