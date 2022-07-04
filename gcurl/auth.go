package gcurl

import (
	"encoding/base64"

	"github.com/golang-jwt/jwt"
	"github.com/lunny/log"
)

type authenticator interface { //BasicAuth, TokenAuth
	HeaderValue() string
}

type BasicAuth struct {
	Username string
	Password string
}

type BearerAuth struct {
	Token string
}

type TokenAuth struct {
	Token string
}

type JWTAuth struct {
	MapClaims jwt.MapClaims
	jwtkey    string
}

func (a *BasicAuth) HeaderValue() string {
	auth := a.Username + ":" + a.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func (a *TokenAuth) HeaderValue() string {
	return "Bearer " + a.Token
}

func (a *BearerAuth) HeaderValue() string {
	return "token " + a.Token
}

func (a *JWTAuth) HeaderValue() string {
	var err error
	//Creating Access Token
	atClaims := jwt.MapClaims{}
	// jwt.StandardClaims
	for k, v := range a.MapClaims {
		atClaims[k] = v
	}
	// atClaims["now"] = time.Now().Unix()
	//	atClaims["exp"] = time.Now().Add(time.Minute * 2).Unix()
	// atClaims["nbf"]
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString([]byte(a.jwtkey))
	if err != nil {
		return ""
	}
	return "Bearer " + token
}

type digestAuth struct {
	*DigestTransport
	Username string
	Password string
}

func applyAuth(r *Request) bool {
	if r.Auth == nil {
		return true
	}

	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}

	switch v := r.Auth.(type) {
	case authenticator: //BasicAuth, TokenAuth
		r.Headers["Authorization"] = v.HeaderValue()
	case string: //req.Auth = "string"
		r.Headers["Authorization"] = v
	case *digestAuth:
		if v.DigestTransport == nil {
			v.DigestTransport = NewDigestTransport(v.Username, v.Password)
		} else {
			v.DigestTransport.Username = v.Username
			v.DigestTransport.Password = v.Password
		}
		r.Client.Transport = v.DigestTransport
	default:
		log.Info("Authen type", v)
		return false
		//		panic(fmt.Errorf("unsupported request.Auth type: %T", v))
	}
	return true
}
