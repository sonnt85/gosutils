package gcurl

import (
	"net/http"
	"net/http/cookiejar"

	"golang.org/x/net/publicsuffix"
)

func applyCookies(req *http.Request, r *Request) {
	if r.Cookies == nil {
		return
	}

	jar := cookieJar(r.Client)
	cookies := jar.Cookies(req.URL)
	for k, v := range r.Cookies {
		cookies = append(cookies, &http.Cookie{Name: k, Value: v})
	}
	jar.SetCookies(req.URL, cookies)
}

func cookieJar(c *http.Client) http.CookieJar {
	if c.Jar == nil {
		options := cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		}
		c.Jar, _ = cookiejar.New(&options)
	}
	return c.Jar
}
