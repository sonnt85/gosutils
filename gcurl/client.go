package gcurl

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

type ConnectionOption struct {
	RequestTimeout      time.Duration
	DialTimeout         time.Duration
	DialKeepAlive       time.Duration
	TLSHandshakeTimeout time.Duration
	InsecureSkipVerify  bool
	ProxyURL            string
	DisableRedirect     bool
}

func NewClient(option *ConnectionOption) (*http.Client, error) {
	transport := newTransport(option)
	if option == nil {
		option = new(ConnectionOption)
		option.RequestTimeout = 30 * time.Second
	}
	if len(option.ProxyURL) != 0 {
		err := setProxyTransport(transport, option.ProxyURL)
		if err != nil {
			return nil, err
		}
	}
	client := &http.Client{
		Timeout:   option.RequestTimeout,
		Transport: transport,
	}

	if option.DisableRedirect {
		client.CheckRedirect = disableRedirect
	}

	return client, nil
}

func newTransport(option *ConnectionOption) *http.Transport {
	if option == nil {
		return NewDefaultTransPort()
	}
	if option.DialKeepAlive == 0 {
		option.DialKeepAlive = 15 * time.Second
	}

	if option.DialTimeout == 0 {
		option.DialTimeout = 30 * time.Second
	}

	if option.TLSHandshakeTimeout == 0 {
		option.TLSHandshakeTimeout = 6 * time.Second
	}

	return &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   option.DialTimeout,
			KeepAlive: option.DialKeepAlive,
		}).Dial,
		TLSHandshakeTimeout: option.TLSHandshakeTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: option.InsecureSkipVerify,
		},
	}
}

func setProxyTransport(transport *http.Transport, proxyURL string) error {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}

	switch u.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(u)
	case "socks5":
		dialer, err := proxy.FromURL(u, proxy.Direct)
		if err != nil {
			return err
		}
		transport.Proxy = http.ProxyFromEnvironment
		transport.Dial = dialer.Dial
	}
	return nil
}

func disableRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func NewDefaultTransPort() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 15 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 6 * time.Second,
		//		ExpectContinueTimeout:  1 * time.Second,
		MaxResponseHeaderBytes: 8192,
		ResponseHeaderTimeout:  time.Millisecond * 5000,
		DisableKeepAlives:      false,
	}
}

func ClientFlushDefaultClient() {
	http.DefaultClient.CloseIdleConnections()
}

func ClientResetDefaultTransport() {
	http.DefaultTransport = NewDefaultTransPort()
}
