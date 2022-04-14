package raggett

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestCookie(t *testing.T) {
	t.Run("Common", func(t *testing.T) {
		c := Cookie("Cookie", "Value").
			Path("/test").
			Domain("example.org").
			ExpiresIn(24 * time.Hour).
			Secure().
			HTTPOnly().
			SameSite(http.SameSiteStrictMode)

		assert.Equal(t, "Cookie=Value; Path=/test; Domain=example.org; Max-Age=86400; HttpOnly; Secure; SameSite=Strict", c.cookie.String())
	})

	t.Run("Expiring", func(t *testing.T) {
		c := Cookie("Cookie", "Value").
			Path("/test").
			Domain("example.org").
			ExpiresNow().
			Secure().
			HTTPOnly().
			SameSite(http.SameSiteStrictMode)

		assert.Equal(t, "Cookie=Value; Path=/test; Domain=example.org; Max-Age=0; HttpOnly; Secure; SameSite=Strict", c.cookie.String())
	})
}
