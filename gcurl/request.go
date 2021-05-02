package gcurl

import (
	"net/http"
	"reflect"
	"strings"
	"time"
	//	"github.com/lunny/log"
)

type Request struct {
	Client        *http.Client
	GlobalHeaders map[string]string
	Headers       map[string]string
	Cookies       map[string]string
	Auth          interface{}
}

func NewRequest(client *http.Client) *Request {
	return &Request{
		Client: client,
	}
}

func (r *Request) Call(method string, url string, body interface{}, timeouts ...time.Duration) (*Response, error) {
	timeout := time.Second * 30
	if len(timeouts) != 0 {
		timeout = timeouts[0]
	}
	payload, err := newPayload(body)
	if err != nil {
		return nil, err
	}

	defer r.reset(payload)

	req, err := http.NewRequest(method, url, payload.reader)

	if err != nil {
		return nil, err
	}

	if r.Client == nil {
		r.Client = new(http.Client)
	}
	r.Client.Timeout = timeout

	applyAuth(r)
	conlen := payload.contentLength
	conlen = 0
	applyHeaders(req, r, payload.contentType, conlen)
	//	log.Warn("payload.contentLength: ", payload.contentLength)
	applyCookies(req, r)
	var resp *http.Response
	if reflect.TypeOf(r.Auth) == reflect.TypeOf(DigestAuth{}) {
		da := r.Auth.(DigestAuth)
		t := da.NewDigestTranport()
		resp, err = t.RoundTrip(req)
	} else {
		resp, err = r.Client.Do(req)
	}

	if err != nil {
		return nil, err
	}

	return &Response{resp, nil}, nil
}

func (r *Request) Get(url string, timeouts ...time.Duration) (*Response, error) {
	return r.Call("GET", url, nil, timeouts...)
}

func (r *Request) GetWithBody(url string, body interface{}, timeouts ...time.Duration) (*Response, error) {
	return r.Call("GET", url, body, timeouts...)
}

func (r *Request) Post(url string, body interface{}, timeouts ...time.Duration) (*Response, error) {
	return r.Call("POST", url, body, timeouts...)
}

func (r *Request) Put(url string, body interface{}, timeouts ...time.Duration) (*Response, error) {
	return r.Call("PUT", url, body, timeouts...)
}

func (r *Request) Patch(url string, body interface{}, timeouts ...time.Duration) (*Response, error) {
	return r.Call("PATCH", url, body, timeouts...)
}

func (r *Request) Delete(url string, timeouts ...time.Duration) (*Response, error) {
	return r.Call("DELETE", url, nil, timeouts...)
}

func (r *Request) Head(url string, timeouts ...time.Duration) (*Response, error) {
	return r.Call("HEAD", url, nil, timeouts...)
}

func (r *Request) Options(url string, timeouts ...time.Duration) (*Response, error) {
	return r.Call("OPTIONS", url, nil, timeouts...)
}

func (r *Request) WithGlobalHeader(name, value string) *Request {
	if r.GlobalHeaders == nil {
		r.GlobalHeaders = make(map[string]string)
	}
	r.GlobalHeaders[name] = value
	return r
}

func (r *Request) WithHeader(name, value string) *Request {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[name] = value
	return r
}

func (r *Request) WithCookie(name, value string) *Request {
	if r.Cookies == nil {
		r.Cookies = make(map[string]string)
	}
	r.Cookies[name] = value
	return r
}

func (r *Request) WithBasicAuth(name, passwd string) *Request {
	r.Auth = &BasicAuth{name, passwd}
	return r
}

func (r *Request) WithTokenAuth(token string) *Request {
	r.Auth = &TokenAuth{token}
	return r
}

func (r *Request) WithDigestAuth(name, password string) *Request {
	r.Auth = &DigestAuth{&DigestTransport{Username: name, Password: password}}
	return r
}

func (r *Request) reset(payload *Payload) {
	r.Headers = nil
	r.Cookies = nil

	if payload.closer != nil {
		payload.closer.Close()
	}
}

func NewURL(u string, query interface{}) string {
	if query == nil {
		return u
	}

	qs := newValues(query)
	if strings.Contains(u, "?") {
		return u + "&" + qs.Encode()
	}
	return u + "?" + qs.Encode()
}
