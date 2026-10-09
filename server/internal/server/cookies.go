package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	accessCookie  = "cs_access"
	refreshCookie = "cs_refresh"
)

// setAuthCookies also sends the tokens as httpOnly cookies for web. Mobile uses the tokens in the response body.
func (h *handler) setAuthCookies(c echo.Context, access string, accessExp time.Time, refresh string, refreshExp time.Time) {
	c.SetCookie(h.cookie(accessCookie, access, accessExp))
	c.SetCookie(h.cookie(refreshCookie, refresh, refreshExp))
}

func (h *handler) clearAuthCookies(c echo.Context) {
	for _, name := range []string{accessCookie, refreshCookie} {
		ck := h.cookie(name, "", time.Unix(0, 0))
		ck.MaxAge = -1
		c.SetCookie(ck)
	}
}

func (h *handler) cookie(name, value string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

// bearerToken returns the token from the Authorization: Bearer header. ok is false if there is no header.
func bearerToken(c echo.Context) (token string, ok bool) {
	h := c.Request().Header.Get(echo.HeaderAuthorization)
	if h == "" {
		return "", false
	}
	const prefix = "bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):]), true
	}
	return "", true // a malformed header still counts as "header present" (it does not fall through to the cookie)
}

func cookieValue(c echo.Context, name string) string {
	ck, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return ck.Value
}
