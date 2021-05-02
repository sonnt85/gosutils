package gcurl

import (
	"net/http"
)

var DefaultUserAgent = "sonnt85/gcurl"

var DefaultHeaders = map[string]string{
	"Connection":      "keep-alive",
	"Accept-Encoding": "gzip, deflate",
	"Accept":          "*/*",
	"User-Agent":      DefaultUserAgent,
}

func applyHeaders(req *http.Request, r *Request, contentType string, contentLength int64) {
	// apply contentType
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	// apply contentLength
	if contentLength > 0 {
		req.ContentLength = contentLength
	}

	// apply custom Headers
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}

	// apply custom global Headers
	for k, v := range r.GlobalHeaders {
		if _, ok := req.Header[k]; !ok {
			req.Header.Set(k, v)
		}
	}

	// apply default headers
	for k, v := range DefaultHeaders {
		if _, ok := req.Header[k]; !ok {
			req.Header.Set(k, v)
		}
	}
}
