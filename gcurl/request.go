package gcurl

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

type Request struct {
	Client        *http.Client      //resued
	GlobalHeaders map[string]string //resued
	Headers       map[string]string
	Cookies       map[string]string
	Auth          interface{}
}

func NewRequest(client *http.Client) *Request {
	return &Request{
		Client: client,
	}
}

//auto reset headers, cookies, and close body (if is closer) affter return
func (r *Request) Call(method string, url string, body interface{}, paras ...interface{}) (*Response, error) {
	timeout := time.Second * 30
	fixSizePayload := false
	if len(paras) != 0 {
		for _, para := range paras {
			switch v := para.(type) {
			case time.Duration:
				timeout = v
			case bool:
				fixSizePayload = v
			default:
			}
		}
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
		if r.Client, err = NewClient(nil); err != nil {
			return nil, err
		}
	}
	r.Client.Timeout = timeout

	if !applyAuth(r) {
		return nil, fmt.Errorf("do not know the Authen type")
	}
	conlen := int64(0)

	if fixSizePayload {
		conlen = payload.contentLength
	}
	applyHeaders(req, r, payload.contentType, conlen)
	//	log.Warn("payload.contentLength: ", payload.contentLength)
	applyCookies(req, r)
	var resp *http.Response
	resp, err = r.Client.Do(req)

	if err != nil {
		return nil, err
	}

	return &Response{resp, nil}, nil
}

func (r *Request) Get(url string, paras ...interface{}) (*Response, error) {
	return r.Call("GET", url, nil, paras...)
}

func (r *Request) GetWithBody(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return r.Call("GET", url, body, paras...)
}

func (r *Request) Post(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return r.Call("POST", url, body, paras...)
}

func (r *Request) Put(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return r.Call("PUT", url, body, paras...)
}

func (r *Request) Patch(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return r.Call("PATCH", url, body, paras...)
}

func (r *Request) Delete(url string, paras ...interface{}) (*Response, error) {
	return r.Call("DELETE", url, nil, paras...)
}

func (r *Request) Head(url string, paras ...interface{}) (*Response, error) {
	return r.Call("HEAD", url, nil, paras...)
}

func (r *Request) Options(url string, paras ...interface{}) (*Response, error) {
	return r.Call("OPTIONS", url, nil, paras...)
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
	r.Auth = &DigestAuth{Username: name, Password: password}
	return r
}

func (r *Request) WithBearerAuth(token string) *Request {
	r.Auth = &BearerAuth{token}
	return r
}

func (r *Request) WithJWTAuth(jwtkey string, mapdata jwt.MapClaims) *Request {
	r.Auth = &JWTAuth{jwtkey: jwtkey, MapClaims: mapdata}
	return r
}

//Support basic,digest, jwt
func (r *Request) WithAuth(auth interface{}) *Request {
	r.Auth = auth
	return r
}

func (r *Request) reset(payload *Payload) {
	r.Headers = nil
	r.Cookies = nil

	if payload.closer != nil {
		payload.closer.Close()
	}
}

//query may be one of the following types:  url.Values, map[string]string, map[string][]string
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
