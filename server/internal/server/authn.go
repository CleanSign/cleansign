package server

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/CleanSign/cleansign/server/internal/shared/authctx"
	"github.com/CleanSign/cleansign/server/internal/shared/errs"
)

// authenticate는 토큰이 있고 유효하면 사용자를 요청 컨텍스트에 담는다. 거절하지는 않는다:
// 로그인이 필요한 곳은 handler가 requireUser로 401을 돌려준다.
// Authorization 헤더가 있으면 그것만 보고, 없을 때만 쿠키를 본다.
func (h *handler) authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token, hasHeader := bearerToken(c)
		if !hasHeader {
			token = cookieValue(c, accessCookie)
		}
		if token == "" {
			return next(c)
		}
		ctx := c.Request().Context()
		userID, sessionID, err := h.Auth.Authenticate(ctx, token)
		switch {
		case err == nil:
			c.SetRequest(c.Request().WithContext(authctx.With(ctx, userID, sessionID)))
		case errors.Is(err, errs.ErrUnauthorized):
			// 만료되었거나 폐기된 토큰: 비로그인으로 처리한다
		default:
			return err
		}
		return next(c)
	}
}

// requireUser는 로그인한 사용자의 ID를 돌려준다. 로그인하지 않았으면 errs.ErrUnauthorized다.
func requireUser(c echo.Context) (string, error) {
	id, ok := authctx.UserID(c.Request().Context())
	if !ok {
		return "", errs.ErrUnauthorized
	}
	return id, nil
}
