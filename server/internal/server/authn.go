package server

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/CleanSign/cleansign/server/internal/shared/authctx"
	"github.com/CleanSign/cleansign/server/internal/shared/errs"
)

// authenticate puts the user in the request context if a token is present and valid. It does not reject:
// where login is required, the handler returns 401 via requireUser.
// If an Authorization header is present only that is checked; the cookie is consulted only when it is absent.
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
			// expired or revoked token: treat as not logged in
		default:
			return err
		}
		return next(c)
	}
}

// requireUser returns the logged-in user's ID. If not logged in, it returns errs.ErrUnauthorized.
func requireUser(c echo.Context) (string, error) {
	id, ok := authctx.UserID(c.Request().Context())
	if !ok {
		return "", errs.ErrUnauthorized
	}
	return id, nil
}
