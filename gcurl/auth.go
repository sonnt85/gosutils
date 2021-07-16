package gcurl

import (
	"encoding/base64"

	"github.com/lunny/log"
)

type authenticator interface { //BasicAuth, TokenAuth
	HeaderValue() string
}

type BasicAuth struct {
	Username string
	Password string
}

type TokenAuth struct {
	Token string
}

func (a *BasicAuth) HeaderValue() string {
	auth := a.Username + ":" + a.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func (a *TokenAuth) HeaderValue() string {
	return "token " + a.Token
}

type DigestAuth struct {
	*DigestTransport
	//	Username string
	//	Password string
}

func (a *DigestAuth) NewDigestTranport() *DigestTransport {
	return NewDigestTransport(a.Username, a.Password)
}

func applyAuth(r *Request) bool {
	if r.Auth == nil {
		return false
	}

	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	//	t := dac.NewTransport(username, password)
	//
	//	resp, err := t.RoundTrip(r)

	switch v := r.Auth.(type) {
	case authenticator: //BasicAuth, TokenAuth
		r.Headers["Authorization"] = v.HeaderValue()
	case string: //req.Auth = "string"
		r.Headers["Authorization"] = v
	case DigestAuth:
		//da := r.Auth.(DigestAuth)
		//t := da.NewDigestTranport()
		//t := dac.NewTransport(a.Username, a.Password)
		//r.Client.Transport

	default:
		log.Info("Authen type", v)

		return false
		//		panic(fmt.Errorf("unsupported request.Auth type: %T", v))
	}
	return true
}
