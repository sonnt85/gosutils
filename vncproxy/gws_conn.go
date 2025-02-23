package vncproxy

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// an adapter for representing WebSocket connection as a net.Conn
// some caveats apply: https://github.com/gorilla/websocket/issues/441

var ErrUnexpectedMessageType = errors.New("unexpected websocket message type")

const (
	pongTimeout  = time.Second * 35
	pingInterval = time.Second * 30
)

type ConnWs struct {
	conn        *websocket.Conn
	readMutex   sync.Mutex
	writeMutex  sync.Mutex
	reader      io.Reader
	messageType int
	stopPingCh  chan struct{}
	pongCh      chan bool
	buff        []byte // buffer for reading - test only
}

func NewConnWs(conn *websocket.Conn) *ConnWs {
	adapter := &ConnWs{
		conn: conn,
	}

	return adapter
}

func (a *ConnWs) Ping1() chan bool {
	if a.pongCh != nil {
		return a.pongCh
	}

	a.stopPingCh = make(chan struct{})
	a.pongCh = make(chan bool)

	timeout := time.AfterFunc(pongTimeout, func() {
		_ = a.Close()
	})

	a.conn.SetPongHandler(func(data string) error {
		timeout.Reset(pongTimeout)

		// non-blocking channel write
		select {
		case a.pongCh <- true:
		default:
		}

		return nil
	})

	// ping loop
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := a.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
					logrus.WithError(err).Error("Failed to write ping message")
				}
			case <-a.stopPingCh:
				return
			}
		}
	}()

	return a.pongCh
}

func (a *ConnWs) Read(b []byte) (n int, err error) {
	// Read() can be called concurrently, and we mutate some internal state here
	a.readMutex.Lock()
	defer a.readMutex.Unlock()
	var messageType int
	if a.reader == nil {
		var rd io.Reader
		messageType, rd, err = a.conn.NextReader()
		if err != nil {
			return 0, err
		}

		if messageType != websocket.BinaryMessage { // && messageType != websocket.TextMessage {
			// reader.Read(p []byte)
			return 0, nil
			// return 0, ErrUnexpectedMessageType
		}

		a.messageType = messageType
		a.reader = rd
	}
	bytesRead, err := a.reader.Read(b)
	if err != nil {
		if f, ok := a.reader.(io.Closer); ok {
			_ = f.Close()
		}
		a.reader = nil
		// EOF for the current Websocket frame, more will probably come so..
		if errors.Is(err, io.EOF) {
			// .. we must hide this from the caller since our semantics are a
			// stream of bytes across many frames
			err = nil
		}
	}
	if a.messageType != websocket.BinaryMessage {
		// fmt.Println("mtype != websocket.BinaryMessage: ", mtype)
		return 0, nil
	}
	return bytesRead, err
}

// Read is not threadsafe though thats okay since there
// should never be more than one reader
func (c *ConnWs) ReadNew(dst []byte) (n int, err error) {
	c.readMutex.Lock()
	defer c.readMutex.Unlock()
	ldst := len(dst)
	//use buffer or read new message
	var src, msg []byte
	var mtype int

	if len(c.buff) > 0 {
		src = c.buff
		c.buff = nil
	} else if mtype, msg, err = c.conn.ReadMessage(); err == nil {
		src = msg
	} else {
		return 0, err
	}
	//copy src->dest
	if len(src) > ldst {
		//copy as much as possible of src into dst
		n = copy(dst, src[:ldst])
		//copy remainder into buffer for next read
		r := src[ldst:]
		lr := len(r)
		c.buff = make([]byte, lr)
		copy(c.buff, r)
	} else {
		//copy all of src into dst
		n = copy(dst, src)
	}
	if mtype != websocket.BinaryMessage {
		// fmt.Println("mtype != websocket.BinaryMessage: ", mtype)
		return 0, nil
	}
	//return bytes copied
	return n, nil
}

func (a *ConnWs) Write(b []byte) (int, error) {
	a.writeMutex.Lock()
	defer a.writeMutex.Unlock()
	// return len(b), a.conn.WriteMessage(websocket.BinaryMessage, b)
	nextWriter, err := a.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	bytesWritten, err := nextWriter.Write(b)
	if err != nil {
		return bytesWritten, err
	}
	err = nextWriter.Close()
	return bytesWritten, err
}

func (a *ConnWs) Close() error {
	select {
	case <-a.stopPingCh:
	default:
		if a.stopPingCh != nil {
			a.stopPingCh <- struct{}{}
			close(a.stopPingCh)
		}
	}
	return a.conn.Close()
}

func (a *ConnWs) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *ConnWs) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *ConnWs) SetDeadline(t time.Time) error {
	if err := a.SetReadDeadline(t); err != nil {
		return err
	}
	return a.SetWriteDeadline(t)
}

func (a *ConnWs) SetReadDeadline(t time.Time) error {
	return a.conn.SetReadDeadline(t)
}

func (a *ConnWs) SetWriteDeadline(t time.Time) error {
	a.writeMutex.Lock()
	defer a.writeMutex.Unlock()
	return a.conn.SetWriteDeadline(t)
}
