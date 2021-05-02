package gcurl

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Response ...
type Response struct {
	*http.Response
	bytes []byte
}

// Content return Response Body as []byte
func (resp *Response) Bytes() ([]byte, error) {
	if resp.bytes != nil {
		return resp.bytes, nil
	}

	var reader io.ReadCloser
	var err error
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		if reader, err = gzip.NewReader(resp.Body); err != nil {
			return nil, err
		}
	case "deflate":
		if reader, err = zlib.NewReader(resp.Body); err != nil {
			return nil, err
		}
	default:
		reader = resp.Body
	}

	defer reader.Close()
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	resp.bytes = b
	return b, nil
}

// Text return Response Body as string
func (resp *Response) Text() (string, error) {
	b, err := resp.Bytes()
	if err != nil {
		return "", nil
	}
	return string(b), nil
}

// OK check Response StatusCode < 400 ?
func (resp *Response) OK() bool {
	return resp.StatusCode < 400
}

// JSON return Response Body as JSON interface{}
func (resp *Response) JSON() (interface{}, error) {
	var v interface{}
	err := resp.JSONUnmarshal(&v)
	return v, err
}

// JSONUnmarshal unmarshal Response Body
func (resp *Response) JSONUnmarshal(data interface{}) error {
	b, err := resp.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, data)
}

// RequestURL return finally request url
func (resp *Response) RequestURL() (*url.URL, error) {
	u := resp.Request.URL
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound,
		http.StatusSeeOther, http.StatusTemporaryRedirect:
		location, err := resp.Location()
		if err != nil {
			return nil, err
		}
		u = u.ResolveReference(location)
	}
	return u, nil
}

// OK check Response StatusCode < 400 ?
func (resp *Response) Close() error {
	return resp.Body.Close()
}
