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
	if option == nil {
		return new(http.Client), nil
	}

	transport := newTransport(option)
	if option.ProxyURL != "" {
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
