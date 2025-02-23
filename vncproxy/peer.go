package vncproxy

import (
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sonnt85/gosutils/bufcopy"

	// "github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var (
	bcopy = bufcopy.New()
)

// Peer represents a vnc proxy Peer
// with a websocket connection and a vnc backend connection
type Source interface {
	RemoteAddr() net.Addr
	io.ReadWriter
	Close() error
}
type Peer struct {
	source      Source //*websocket.Conn
	target      *net.Conn
	localListen net.Listener
	Token       string
	ConnectAt   time.Time
}

func isNewUnixSocket(addr string) bool {
	fileInfo, err := os.Stat(addr)
	if err != nil {
		return false // Assume addr is a TCP socket address
	}
	return fileInfo.Mode()&os.ModeSocket != 0
}

func NewPeer(ws Source, addr, token string, localListen ...bool) (*Peer, error) {
	if ws == nil {
		return nil, errors.New("websocket connection is nil")
	}
	var c net.Conn
	var err error
	var l net.Listener
	localL := len(localListen) > 0 && localListen[0]
	if isNewUnixSocket(addr) {
		c, err = net.DialTimeout("unix", addr, 5*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "cannot connect to unix vnc backend")
		}
	} else if localL {
		l, err = net.Listen("tcp", addr)
		if err != nil {
			return nil, errors.Wrap(err, "cannot listen on address")
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
	if !localL {
		err = c.(*net.TCPConn).SetKeepAlive(true)
		if err != nil {
			return nil, errors.Wrap(err, "enable vnc backend connection keepalive failed")
		}

		err = c.(*net.TCPConn).SetKeepAlivePeriod(30 * time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "set vnc backend connection keepalive period failed")
		}
	}
	return &Peer{
		source:      ws,
		target:      &c,
		localListen: l,
		Token:       token,
		ConnectAt:   time.Now(),
	}, nil
}

// ReadSource copy source stream to target connection
func (p *Peer) ReadSource() error {
	target := *p.target
	if _, err := bcopy.Copy(target, p.source); err != nil {
		return errors.Wrapf(err, "copy source(%v) => target(%v) failed", p.source.RemoteAddr(), target.RemoteAddr())
	}
	return nil
}

func (p *Peer) ReadRawSource() error {
	target := *p.target
	if _, err := bcopy.Copy(target, p.source); err != nil {
		return errors.Wrapf(err, "copy source(%v) => target(%v) failed", p.source.RemoteAddr(), target.RemoteAddr())
	}
	return nil
}

// ReadTarget copys target stream to source connection
func (p *Peer) ReadTarget() error {
	target := *p.target
	if _, err := bcopy.Copy(p.source, target); err != nil {
		return errors.Wrapf(err, "copy target(%v) => source(%v) failed", target.RemoteAddr(), p.source.RemoteAddr())
	}
	return nil
}

func (p *Peer) ReadRawTarget() error {
	target := *p.target
	if _, err := bcopy.Copy(p.source, target); err != nil {
		return errors.Wrapf(err, "copy target(%v) => source(%v) failed", target.RemoteAddr(), p.source.RemoteAddr())
	}
	return nil
}

// Close close the websocket connection and the vnc backend connection
func (p *Peer) Close() {
	p.source.Close()
	if p.target == nil {
		(*p.target).Close()
	}
	if p.localListen != nil {
		p.localListen.Close()
	}
}

func (peer *Peer) StartCopy() (err error) {
	if peer.localListen != nil {
		var conn net.Conn
		conn, err = peer.localListen.Accept()
		if err != nil {
			return err
		}
		// log.Infof("Accept new connection from %v", conn.RemoteAddr())
		defer func() {
			conn.Close()
			// log.Infof("Close connection from %v", conn.RemoteAddr())
		}()
		_, err = bcopy.Copy2Way(conn, peer.source)
		return
	} else {
		go func() {
			if err := peer.ReadTarget(); err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				return
			}
		}()

		if err = peer.ReadSource(); err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			return
		}
		return
	}
}

func (peer *Peer) StartRawCopy() (err error) {
	if peer.localListen != nil {
		var conn net.Conn
		conn, err = peer.localListen.Accept()
		if err != nil {
			return err
		}
		// log.Infof("Accept new connection from %v", conn.RemoteAddr())
		defer func() {
			conn.Close()
			// log.Infof("Close connection from %v", conn.RemoteAddr())
		}()
		_, err = bcopy.Copy2Way(conn, peer.source)
		return
	} else {
		go func() {
			if err := peer.ReadRawTarget(); err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				return
			}
		}()

		if err = peer.ReadRawSource(); err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			return
		}
	}
	return
}
