package vncproxy

import (
	"net"
	"os"
	"time"

	"github.com/sonnt85/gosutils/bufcopy"

	// "github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
)

var (
	bcopy = bufcopy.New()
)

// Peer represents a vnc proxy Peer
// with a websocket connection and a vnc backend connection
type Peer struct {
	source    *websocket.Conn
	target    net.Conn
	Token     string
	ConnectAt time.Time
}

func isNewUnixSocket(addr string) bool {
	fileInfo, err := os.Stat(addr)
	if err != nil {
		return false // Assume addr is a TCP socket address
	}
	return fileInfo.Mode()&os.ModeSocket != 0
}

func NewPeer(ws *websocket.Conn, addr, token string) (*Peer, error) {
	if ws == nil {
		return nil, errors.New("websocket connection is nil")
	}
	var c net.Conn
	var err error
	if isNewUnixSocket(addr) {
		c, err = net.DialTimeout("unix", addr, 5*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cannot connect to unix vnc backend")
		}
	} else {
		c, err = net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cannot connect to vnc backend")
		}
	}
	// c, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to vnc backend")
	}

	err = c.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, errors.Wrap(err, "enable vnc backend connection keepalive failed")
	}

	err = c.(*net.TCPConn).SetKeepAlivePeriod(30 * time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "set vnc backend connection keepalive period failed")
	}

	return &Peer{
		source:    ws,
		target:    c,
		Token:     token,
		ConnectAt: time.Now(),
	}, nil
}

// ReadSource copy source stream to target connection
func (p *Peer) ReadSource() error {
	if _, err := bcopy.Copy(p.target, p.source); err != nil {
		return errors.Wrapf(err, "copy source(%v) => target(%v) failed", p.source.RemoteAddr(), p.target.RemoteAddr())
	}
	return nil
}

// ReadTarget copys target stream to source connection
func (p *Peer) ReadTarget() error {
	if _, err := bcopy.Copy(p.source, p.target); err != nil {
		return errors.Wrapf(err, "copy target(%v) => source(%v) failed", p.target.RemoteAddr(), p.source.RemoteAddr())
	}
	return nil
}

// Close close the websocket connection and the vnc backend connection
func (p *Peer) Close() {
	p.source.Close()
	p.target.Close()
}
