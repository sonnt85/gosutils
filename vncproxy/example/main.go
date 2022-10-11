package main

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/vncproxy"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

func main() {
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

func NewVNCProxy() *vncproxy.Proxy {
	return vncproxy.New(&vncproxy.Config{
		Logger: logrus.StandardLogger(),
		TokenHandler: func(r *http.Request, retToken ...*string) (addr string, err error) {
			return ":5901", nil
		},
		OnConnectFunc: func(*vncproxy.Peer) {
		},
		OnDisconectFunc: func(*vncproxy.Peer) {
		},
	})
}
