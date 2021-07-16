package gcurl

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	algorithmMD5        = "MD5"
	algorithmMD5Sess    = "MD5-SESS"
	algorithmSHA256     = "SHA-256"
	algorithmSHA256Sess = "SHA-256-SESS"
)

type authorization struct {
	Algorithm string // unquoted
	Cnonce    string // quoted
	Nc        int    // unquoted
	Nonce     string // quoted
	Opaque    string // quoted
	Qop       string // unquoted
	Realm     string // quoted
	Response  string // quoted
	URI       string // quoted
	Userhash  bool   // quoted
	Username  string // quoted
	Username_ string // quoted
}

type DigestRequest struct {
	Body       string
	Method     string
	Password   string
	URI        string
	Username   string
	Header     http.Header
	Auth       *authorization
	Wa         *wwwAuthenticate
	CertVal    bool
	HTTPClient *http.Client
}

type DigestTransport struct {
	Password   string
	Username   string
	HTTPClient *http.Client
}

type wwwAuthenticate struct {
	Algorithm string // unquoted
	Domain    string // quoted
	Nonce     string // quoted
	Opaque    string // quoted
	Qop       string // quoted
	Realm     string // quoted
	Stale     bool   // unquoted
	Charset   string // quoted
	Userhash  bool   // quoted
}

// NewRequest creates a new DigestRequest object
func NewDigestRequest(username, password, method, uri, body string) DigestRequest {
	dr := DigestRequest{}
	dr.UpdateRequest(username, password, method, uri, body)
	dr.CertVal = true
	return dr
}

// NewTransport creates a new DigestTransport object
func NewDigestTransport(username, password string) *DigestTransport {
	dt := DigestTransport{}
	dt.Password = password
	dt.Username = username
	return &dt
}

func (dr *DigestRequest) getHTTPClient() *http.Client {
	if dr.HTTPClient != nil {
		return dr.HTTPClient
	}
	tlsConfig := tls.Config{}
	if !dr.CertVal {
		tlsConfig.InsecureSkipVerify = true
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
		},
	}
}

// UpdateRequest is called when you want to reuse an existing
//  DigestRequest connection with new request information
func (dr *DigestRequest) UpdateRequest(username, password, method, uri, body string) *DigestRequest {
	dr.Body = body
	dr.Method = method
	dr.Password = password
	dr.URI = uri
	dr.Username = username
	dr.Header = make(map[string][]string)
	return dr
}

// RoundTrip implements the http.RoundTripper interface
func (dt *DigestTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	username := dt.Username
	password := dt.Password
	method := req.Method
	uri := req.URL.String()

	var body string
	if req.Body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(req.Body)
		body = buf.String()
	}

	dr := NewDigestRequest(username, password, method, uri, body)
	if dt.HTTPClient != nil {
		dr.HTTPClient = dt.HTTPClient
	}

	return dr.Execute()
}

// Execute initialise the request and get a response
func (dr *DigestRequest) Execute() (resp *http.Response, err error) {

	if dr.Auth != nil {
		return dr.executeExistingDigest()
	}

	var req *http.Request
	if req, err = http.NewRequest(dr.Method, dr.URI, bytes.NewReader([]byte(dr.Body))); err != nil {
		return nil, err
	}
	req.Header = dr.Header

	client := dr.getHTTPClient()

	if resp, err = client.Do(req); err != nil {
		return nil, err
	}

	if resp.StatusCode == 401 {
		return dr.executeNewDigest(resp)
	}

	// return the resp to user to handle resp.body.Close()
	return resp, nil
}

func (dr *DigestRequest) executeNewDigest(resp *http.Response) (resp2 *http.Response, err error) {
	var (
		auth     *authorization
		wa       *wwwAuthenticate
		waString string
	)

	// body not required for authentication, closing
	resp.Body.Close()

	if waString = resp.Header.Get("WWW-Authenticate"); waString == "" {
		return nil, fmt.Errorf("failed to get WWW-Authenticate header, please check your server configuration")
	}
	wa = newWwwAuthenticate(waString)
	dr.Wa = wa

	if auth, err = newAuthorization(dr); err != nil {
		return nil, err
	}

	if resp2, err = dr.executeRequest(auth.toString()); err != nil {
		return nil, err
	}

	dr.Auth = auth
	return resp2, nil
}

func (dr *DigestRequest) executeExistingDigest() (resp *http.Response, err error) {
	var auth *authorization

	if auth, err = dr.Auth.refreshAuthorization(dr); err != nil {
		return nil, err
	}
	dr.Auth = auth

	return dr.executeRequest(dr.Auth.toString())
}

func (dr *DigestRequest) executeRequest(authString string) (resp *http.Response, err error) {
	var req *http.Request

	if req, err = http.NewRequest(dr.Method, dr.URI, bytes.NewReader([]byte(dr.Body))); err != nil {
		return nil, err
	}
	req.Header = dr.Header
	req.Header.Add("Authorization", authString)

	client := dr.getHTTPClient()
	return client.Do(req)
}

func newAuthorization(dr *DigestRequest) (*authorization, error) {

	ah := authorization{
		Algorithm: dr.Wa.Algorithm,
		Cnonce:    "",
		Nc:        0,
		Nonce:     dr.Wa.Nonce,
		Opaque:    dr.Wa.Opaque,
		Qop:       "",
		Realm:     dr.Wa.Realm,
		Response:  "",
		URI:       "",
		Userhash:  dr.Wa.Userhash,
		Username:  "",
		Username_: "", // TODO
	}

	return ah.refreshAuthorization(dr)
}

func (ah *authorization) refreshAuthorization(dr *DigestRequest) (*authorization, error) {

	ah.Username = dr.Username

	if ah.Userhash {
		ah.Username = ah.hash(fmt.Sprintf("%s:%s", ah.Username, ah.Realm))
	}

	ah.Nc++

	ah.Cnonce = ah.hash(fmt.Sprintf("%d:%s:my_value", time.Now().UnixNano(), dr.Username))

	url, err := url.Parse(dr.URI)
	if err != nil {
		return nil, err
	}

	ah.URI = url.RequestURI()
	ah.Response = ah.computeResponse(dr)

	return ah, nil
}

func (ah *authorization) computeResponse(dr *DigestRequest) (s string) {

	kdSecret := ah.hash(ah.computeA1(dr))
	kdData := fmt.Sprintf("%s:%08x:%s:%s:%s", ah.Nonce, ah.Nc, ah.Cnonce, ah.Qop, ah.hash(ah.computeA2(dr)))

	return ah.hash(fmt.Sprintf("%s:%s", kdSecret, kdData))
}

func (ah *authorization) computeA1(dr *DigestRequest) string {

	algorithm := strings.ToUpper(ah.Algorithm)

	if algorithm == "" || algorithm == algorithmMD5 || algorithm == algorithmSHA256 {
		return fmt.Sprintf("%s:%s:%s", ah.Username, ah.Realm, dr.Password)
	}

	if algorithm == algorithmMD5Sess || algorithm == algorithmSHA256Sess {
		upHash := ah.hash(fmt.Sprintf("%s:%s:%s", ah.Username, ah.Realm, dr.Password))
		return fmt.Sprintf("%s:%s:%s", upHash, ah.Nonce, ah.Cnonce)
	}

	return ""
}

func (ah *authorization) computeA2(dr *DigestRequest) string {

	if strings.Contains(dr.Wa.Qop, "auth-int") {
		ah.Qop = "auth-int"
		return fmt.Sprintf("%s:%s:%s", dr.Method, ah.URI, ah.hash(dr.Body))
	}

	if dr.Wa.Qop == "auth" || dr.Wa.Qop == "" {
		ah.Qop = "auth"
		return fmt.Sprintf("%s:%s", dr.Method, ah.URI)
	}

	return ""
}

func (ah *authorization) hash(a string) string {
	var h hash.Hash
	algorithm := strings.ToUpper(ah.Algorithm)

	if algorithm == "" || algorithm == algorithmMD5 || algorithm == algorithmMD5Sess {
		h = md5.New()
	} else if algorithm == algorithmSHA256 || algorithm == algorithmSHA256Sess {
		h = sha256.New()
	} else {
		// unknown algorithm
		return ""
	}

	io.WriteString(h, a)
	return hex.EncodeToString(h.Sum(nil))
}

func (ah *authorization) toString() string {
	var buffer bytes.Buffer

	buffer.WriteString("Digest ")

	if ah.Username != "" {
		buffer.WriteString(fmt.Sprintf("username=\"%s\", ", ah.Username))
	}

	if ah.Realm != "" {
		buffer.WriteString(fmt.Sprintf("realm=\"%s\", ", ah.Realm))
	}

	if ah.Nonce != "" {
		buffer.WriteString(fmt.Sprintf("nonce=\"%s\", ", ah.Nonce))
	}

	if ah.URI != "" {
		buffer.WriteString(fmt.Sprintf("uri=\"%s\", ", ah.URI))
	}

	if ah.Response != "" {
		buffer.WriteString(fmt.Sprintf("response=\"%s\", ", ah.Response))
	}

	if ah.Algorithm != "" {
		buffer.WriteString(fmt.Sprintf("algorithm=%s, ", ah.Algorithm))
	}

	if ah.Cnonce != "" {
		buffer.WriteString(fmt.Sprintf("cnonce=\"%s\", ", ah.Cnonce))
	}

	if ah.Opaque != "" {
		buffer.WriteString(fmt.Sprintf("opaque=\"%s\", ", ah.Opaque))
	}

	if ah.Qop != "" {
		buffer.WriteString(fmt.Sprintf("qop=%s, ", ah.Qop))
	}

	if ah.Nc != 0 {
		buffer.WriteString(fmt.Sprintf("nc=%08x, ", ah.Nc))
	}

	if ah.Userhash {
		buffer.WriteString("userhash=true, ")
	}

	s := buffer.String()

	return strings.TrimSuffix(s, ", ")
}

func newWwwAuthenticate(s string) *wwwAuthenticate {

	var wa = wwwAuthenticate{}

	algorithmRegex := regexp.MustCompile(`algorithm="([^ ,]+)"`)
	algorithmMatch := algorithmRegex.FindStringSubmatch(s)
	if algorithmMatch != nil {
		wa.Algorithm = algorithmMatch[1]
	}

	domainRegex := regexp.MustCompile(`domain="(.+?)"`)
	domainMatch := domainRegex.FindStringSubmatch(s)
	if domainMatch != nil {
		wa.Domain = domainMatch[1]
	}

	nonceRegex := regexp.MustCompile(`nonce="(.+?)"`)
	nonceMatch := nonceRegex.FindStringSubmatch(s)
	if nonceMatch != nil {
		wa.Nonce = nonceMatch[1]
	}

	opaqueRegex := regexp.MustCompile(`opaque="(.+?)"`)
	opaqueMatch := opaqueRegex.FindStringSubmatch(s)
	if opaqueMatch != nil {
		wa.Opaque = opaqueMatch[1]
	}

	qopRegex := regexp.MustCompile(`qop="(.+?)"`)
	qopMatch := qopRegex.FindStringSubmatch(s)
	if qopMatch != nil {
		wa.Qop = qopMatch[1]
	}

	realmRegex := regexp.MustCompile(`realm="(.+?)"`)
	realmMatch := realmRegex.FindStringSubmatch(s)
	if realmMatch != nil {
		wa.Realm = realmMatch[1]
	}

	staleRegex := regexp.MustCompile(`stale=([^ ,])"`)
	staleMatch := staleRegex.FindStringSubmatch(s)
	if staleMatch != nil {
		wa.Stale = (strings.ToLower(staleMatch[1]) == "true")
	}

	charsetRegex := regexp.MustCompile(`charset="(.+?)"`)
	charsetMatch := charsetRegex.FindStringSubmatch(s)
	if charsetMatch != nil {
		wa.Charset = charsetMatch[1]
	}

	userhashRegex := regexp.MustCompile(`userhash=([^ ,])"`)
	userhashMatch := userhashRegex.FindStringSubmatch(s)
	if userhashMatch != nil {
		wa.Userhash = (strings.ToLower(userhashMatch[1]) == "true")
	}

	return &wa
}
