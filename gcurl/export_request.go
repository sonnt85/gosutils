package gcurl

var defaultRequest = NewRequest(nil)

func GetDefaultRequest() *Request {
	return defaultRequest
}

func Call(method string, url string, body interface{}, paras ...interface{}) (*Response, error) {
	return defaultRequest.Call(method, url, body, paras...)
}

func Get(url string, paras ...interface{}) (*Response, error) {
	return defaultRequest.Get(url, paras...)
}

func GetWithBody(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return defaultRequest.GetWithBody(url, body, paras...)
}

func Post(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return defaultRequest.Post(url, body, paras...)
}

func Put(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return defaultRequest.Put(url, body, paras...)
}

func Patch(url string, body interface{}, paras ...interface{}) (*Response, error) {
	return defaultRequest.Patch(url, body, paras...)
}

func Delete(url string, paras ...interface{}) (*Response, error) {
	return defaultRequest.Delete(url, paras...)
}

func Head(url string, paras ...interface{}) (*Response, error) {
	return defaultRequest.Head(url, paras...)
}

func Options(url string, paras ...interface{}) (*Response, error) {
	return defaultRequest.Options(url, paras...)
}

func WithGlobalHeader(name, value string) *Request {
	return defaultRequest.WithGlobalHeader(name, value)
}

func WithHeader(name, value string) *Request {
	return defaultRequest.WithHeader(name, value)
}

func WithCookie(name, value string) *Request {
	return defaultRequest.WithCookie(name, value)
}

func WithBasicAuth(name, passwd string) *Request {
	return defaultRequest.WithBasicAuth(name, passwd)
}

func WithTokenAuth(token string) *Request {
	return defaultRequest.WithTokenAuth(token)
}

func WithDigestAuth(name, password string) *Request {
	return defaultRequest.WithDigestAuth(name, password)
}
