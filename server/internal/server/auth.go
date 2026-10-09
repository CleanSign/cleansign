package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/CleanSign/cleansign/server/api"
	"github.com/CleanSign/cleansign/server/internal/domain/auth"
)

const maxJSONBody = 1 << 20

// decodeJSON은 JSON 본문을 읽는다. 읽지 못하면 요청 형식 오류다.
func decodeJSON(c echo.Context, dst any) error {
	body := http.MaxBytesReader(c.Response(), c.Request().Body, maxJSONBody)
	if err := json.NewDecoder(body).Decode(dst); err != nil {
		return errBadBody
	}
	return nil
}

func (h *handler) SendEmailCode(c echo.Context) error {
	var body api.SendEmailCodeRequest
	if err := decodeJSON(c, &body); err != nil {
		return err
	}
	wait, err := h.Auth.SendCode(c.Request().Context(), body.Email)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.SendEmailCodeResponse{RetryAfterSeconds: int(wait.Seconds())})
}

func (h *handler) ConfirmEmailCode(c echo.Context) error {
	var body api.ConfirmEmailCodeRequest
	if err := decodeJSON(c, &body); err != nil {
		return err
	}
	remember := body.RememberDevice != nil && *body.RememberDevice
	res, err := h.Auth.Confirm(c.Request().Context(), body.Email, strings.TrimSpace(body.Code), remember)
	if err != nil {
		return err
	}
	return h.respondTokens(c, res.Tokens)
}

func (h *handler) RefreshTokens(c echo.Context) error {
	// 본문은 비어 있어도 된다(web은 쿠키로 보낸다)
	var body api.RefreshRequest
	if err := json.NewDecoder(io.LimitReader(c.Request().Body, maxJSONBody)).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		return errBadBody
	}
	token := cookieValue(c, refreshCookie)
	if body.RefreshToken != nil && *body.RefreshToken != "" {
		token = *body.RefreshToken
	}
	tokens, err := h.Auth.Refresh(c.Request().Context(), token)
	if err != nil {
		return err
	}
	return h.respondTokens(c, tokens)
}

// Logout은 가진 토큰이 가리키는 세션을 폐기하고 쿠키를 지운다.
func (h *handler) Logout(c echo.Context) error {
	ctx := c.Request().Context()
	tokens := []string{cookieValue(c, accessCookie), cookieValue(c, refreshCookie)}
	if bearer, ok := bearerToken(c); ok {
		tokens = append(tokens, bearer)
	}
	for _, t := range tokens {
		if err := h.Auth.Logout(ctx, t); err != nil {
			return err
		}
	}
	h.clearAuthCookies(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *handler) respondTokens(c echo.Context, t auth.Tokens) error {
	h.setAuthCookies(c, t.Access, t.AccessExpiresAt, t.Refresh, t.RefreshExpiresAt)
	return c.JSON(http.StatusOK, api.Tokens{
		AccessToken:      t.Access,
		RefreshToken:     t.Refresh,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshExpiresAt: t.RefreshExpiresAt,
	})
}
