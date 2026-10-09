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

// setAuthCookies는 web을 위해 토큰을 httpOnly 쿠키로도 내려 준다. 모바일은 응답 본문의 토큰을 쓴다.
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

// bearerToken은 Authorization: Bearer 헤더의 토큰을 돌려준다. 헤더가 없으면 ok가 false다.
func bearerToken(c echo.Context) (token string, ok bool) {
	h := c.Request().Header.Get(echo.HeaderAuthorization)
	if h == "" {
		return "", false
	}
	const prefix = "bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):]), true
	}
	return "", true // 형식이 틀린 헤더도 "헤더가 있음"으로 본다(쿠키로 넘어가지 않는다)
}

func cookieValue(c echo.Context, name string) string {
	ck, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return ck.Value
}
