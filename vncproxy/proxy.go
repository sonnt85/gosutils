package vncproxy

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	// log "github.com/sirupsen/logrus"

	"github.com/sonnt85/gosutils/endec"
	"github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/sregexp"
	"golang.org/x/net/websocket"
	// "github.com/gorilla/websocket"
)

type TokenHandler func(r *http.Request, retToken ...*string) (addr string, err error)

// Config represents vnc proxy config
type Config struct {
	Logger slogrus.Logger
	TokenHandler
	OnConnectFunc   func(*Peer)
	OnDisconectFunc func(*Peer)
}

// Proxy represents vnc proxy
type Proxy struct {
	logger slogrus.Logger
	// peernew         hashmap.MapEmpty[*Peer]
	peers           map[*Peer]struct{}
	l               sync.RWMutex
	tokenHandler    TokenHandler
	OnConnectFunc   func(*Peer)
	OnDisconectFunc func(*Peer)
}

// New returns a vnc proxy
// If token handler is nil, vnc backend address will always be :5901
func New(conf *Config) *Proxy {
	if conf.TokenHandler == nil {
		conf.TokenHandler = func(r *http.Request, retToken ...*string) (addr string, err error) {
			return ":5901", nil
		}
	}

	return &Proxy{
		logger:          conf.Logger,
		peers:           make(map[*Peer]struct{}),
		l:               sync.RWMutex{},
		tokenHandler:    conf.TokenHandler,
		OnConnectFunc:   conf.OnConnectFunc,
		OnDisconectFunc: conf.OnDisconectFunc,
	}
}

// ServeWS provides websocket handler
func (p *Proxy) ServeWS(ws *websocket.Conn) {
	p.logger.Debugf("ServeWS")
	ws.PayloadType = websocket.BinaryFrame

	r := ws.Request()
	p.logger.Debugf("request url: %v", r.URL)

	// get vnc backend server addr
	retToken := ""
	addr, err := p.tokenHandler(r, &retToken)
	if err != nil {
		p.logger.Infof("get vnc backend failed: %v", err)
		return
	}

	peer, err := NewPeer(ws, addr, retToken)
	if err != nil {
		p.logger.Infof("new vnc peer failed: %v", err)
		return
	}

	if p.OnConnectFunc != nil {
		p.OnConnectFunc(peer)
	}
	p.addPeer(peer)
	defer func() {
		if p.OnDisconectFunc != nil {
			p.OnDisconectFunc(peer)
		}
		p.logger.Info("close peer")
		p.deletePeer(peer)
	}()

	go func() {
		if err := peer.ReadTarget(); err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			p.logger.Info(err)
			return
		}
	}()

	if err = peer.ReadSource(); err != nil {
		if strings.Contains(err.Error(), "use of closed network connection") {
			return
		}
		p.logger.Info(err)
		return
	}
}

func (p *Proxy) addPeer(peer *Peer) {
	p.l.Lock()
	p.peers[peer] = struct{}{}
	p.l.Unlock()
}

func (p *Proxy) deletePeer(peer *Peer) {
	p.l.Lock()
	delete(p.peers, peer)
	peer.Close()
	p.l.Unlock()
}

func (p *Proxy) Peers() map[*Peer]struct{} {
	p.l.RLock()
	defer p.l.RUnlock()
	return p.peers
}

func (p *Proxy) GetConnectionTimeRegexp(tokenRegexp string) (d map[string]time.Duration, isConnected bool) {
	ps := p.Peers()
	d = make(map[string]time.Duration, len(ps))
	var ok bool
	var token string
	for v, _ := range ps {
		if sregexp.New(tokenRegexp).MatchString(v.Token) {
			_, ok = d[v.Token]
			if ok {
				token = fmt.Sprintf("%s-%d", v.Token, endec.RandInt())
			} else {
				token = v.Token
			}
			d[token] = time.Since(v.ConnectAt)
			isConnected = true
		}
	}
	return
}

func (p *Proxy) GetMaxConnectionTime(token string) (d time.Duration, isConnected bool) {
	var lastD time.Duration
	for v, _ := range p.Peers() {
		if token == v.Token {
			lastD = time.Since(v.ConnectAt)
			if d < lastD {
				d = lastD
			}
			isConnected = true
		}
	}
	return
}

func appendTokens(oldtokens []string, addtoken string, num int) (rettokens []string) {
	if num == 0 {
		return oldtokens
	} else if num == 1 {
		rettokens = append(oldtokens, addtoken)
	} else {
		rettokens = append(oldtokens, fmt.Sprintf("%s [%d]", addtoken, num))
	}
	return
}

func (p *Proxy) Tokens() (rettokens []string) {
	var tmpRet []string
	p.l.RLock()
	for i, _ := range p.peers {
		tmpRet = append(tmpRet, i.Token)
	}
	p.l.RUnlock()
	sort.Strings(tmpRet)
	cnt := 0
	lastToken := ""
	isLast := false
	if len(tmpRet) == 1 {
		return []string{tmpRet[0]}
	}
	for i := 0; i < len(tmpRet); i++ {
		isLast = (i == len(tmpRet)-1)
		if isLast {
			if cnt == 0 {
				rettokens = appendTokens(rettokens, tmpRet[i], 1)
			} else {
				if tmpRet[i] == lastToken {
					rettokens = appendTokens(rettokens, lastToken, cnt+1)
				} else {
					rettokens = appendTokens(rettokens, lastToken, cnt)
					rettokens = appendTokens(rettokens, tmpRet[i], 1)
				}
			}
			return
		} else {
			if cnt == 0 {
				lastToken = tmpRet[i]
				cnt = 1
			} else {
				if tmpRet[i] != lastToken {
					rettokens = appendTokens(rettokens, lastToken, cnt)
					cnt = 1
					lastToken = tmpRet[i]
				} else {
					cnt++
				}
			}
		}
	}
	return
}
