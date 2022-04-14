package raggett

import (
	"net/http"
	"time"
)

// Cookie creates a new basic cookie with the provided name and value.
// The returned value can be use chained methods from ChainedCookie in order
// to set other cookie parameters. See ChainedCookie's documentation for further
// information.
// The result of this method (and chaining) can be provided directly to your
// request object. For instance:
//
//  func HandleSomething(req MyRequestObject) error {
//		req.AddCookie(raggett.Cookie("name", "value").HTTPOnly().Secure())
//      // ...
//  }
func Cookie(name, value string) *ChainedCookie {
	return &ChainedCookie{
		cookie: http.Cookie{Name: name, Value: value},
	}
}

type ChainedCookie struct {
	cookie http.Cookie
}

// Path sets the cookie's path component.
// See more: https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#define_where_cookies_are_sent
func (c *ChainedCookie) Path(path string) *ChainedCookie {
	c.cookie.Path = path
	return c
}

// Domain sets the cookie's domain component.
// See more: https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#define_where_cookies_are_sent
func (c *ChainedCookie) Domain(domain string) *ChainedCookie {
	c.cookie.Domain = domain
	return c
}

// ExpiresIn sets the lifetime duration for this cookie. For instance, for a
// cookie to automatically expire in a day, use 24 * time.Hour
// See More: https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#define_the_lifetime_of_a_cookie
func (c *ChainedCookie) ExpiresIn(duration time.Duration) *ChainedCookie {
	c.cookie.MaxAge = int(duration.Seconds())
	return c
}

// ExpiresNow is a convenience method that marks this cookie as already expired.
// This can be used to immediately "delete" a cookie from the browser.
func (c *ChainedCookie) ExpiresNow() *ChainedCookie {
	c.cookie.MaxAge = -1
	return c
}

// Secure marks the cookie with the Secure flag. A cookie with the Secure
// attribute is only sent to the server with an encrypted request over the HTTPS
// protocol. It's never sent with unsecured HTTP (except on localhost), which
// means man-in-the-middle attackers can't access it easily. Insecure sites
// (with http: in the URL) can't set cookies with the Secure attribute.
func (c *ChainedCookie) Secure() *ChainedCookie {
	c.cookie.Secure = true
	return c
}

// HTTPOnly sets the cookie as HTTPOnly, rendering it inaccessible to the
// JavaScript Document.cookie API; it's only sent to the server. For example,
// cookies that persist in server-side sessions don't need to be available to
// JavaScript and should have the HttpOnly attribute. This precaution helps
// mitigate cross-site scripting (XSS) attacks.
func (c *ChainedCookie) HTTPOnly() *ChainedCookie {
	c.cookie.HttpOnly = true
	return c
}

// SameSite declares whether the cookie should be restricted to a first-party or
// same-site context.
// See more: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite
func (c *ChainedCookie) SameSite(sameSite http.SameSite) *ChainedCookie {
	c.cookie.SameSite = sameSite
	return c
}
