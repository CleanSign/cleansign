package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/CleanSign/cleansign/server/internal/domain/auth"
	"github.com/CleanSign/cleansign/server/internal/domain/contract"
	"github.com/CleanSign/cleansign/server/internal/domain/user"
	"github.com/CleanSign/cleansign/server/internal/shared/errs"
)

// httpError는 요청 형식 오류처럼 HTTP 계층에서만 생기는 오류다.
type httpError struct {
	status  int
	message string
}

func (e *httpError) Error() string { return e.message }

func badRequest(message string) error { return &httpError{http.StatusBadRequest, message} }

var errBadBody = badRequest("요청 형식이 올바르지 않아요")

const internalMessage = "서버에 문제가 생겼어요"

// errorHandler는 모든 오류를 { "error": "<메시지>" } 모양의 응답으로 바꾼다.
func (h *handler) errorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	status, message := classify(err)
	if status == http.StatusInternalServerError {
		h.Log.Error("request failed", "method", c.Request().Method, "path", c.Request().URL.Path, "error", err)
	}

	var retry *auth.RetryAfterError
	if errors.As(err, &retry) {
		secs := int((retry.After + 999_999_999) / 1_000_000_000) // 올림
		c.Response().Header().Set(echo.HeaderRetryAfter, strconv.Itoa(max(secs, 1)))
	}
	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(status)
		return
	}
	if err := c.JSON(status, map[string]string{"error": message}); err != nil {
		h.Log.Error("write error response", "error", err)
	}
}

// classify는 오류를 HTTP 상태와 사용자에게 보여 줄 메시지로 바꾼다.
func classify(err error) (int, string) {
	var (
		he    *httpError
		retry *auth.RetryAfterError
		echoE *echo.HTTPError
	)
	switch {
	case errors.As(err, &he):
		return he.status, he.message
	case errors.Is(err, errs.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, errs.ErrNotFound):
		return http.StatusNotFound, "not found"

	case errors.Is(err, contract.ErrFileRequired):
		return http.StatusBadRequest, contract.ErrFileRequired.Error()
	case errors.Is(err, contract.ErrNotPDF):
		return http.StatusUnsupportedMediaType, contract.ErrNotPDF.Error()
	case errors.Is(err, contract.ErrBadSize):
		return http.StatusRequestEntityTooLarge, contract.ErrBadSize.Error()

	case errors.Is(err, user.ErrInvalidEmail):
		return http.StatusBadRequest, user.ErrInvalidEmail.Error()
	case errors.Is(err, user.ErrConsentRequired):
		return http.StatusBadRequest, user.ErrConsentRequired.Error()

	case errors.As(err, &retry):
		return http.StatusTooManyRequests, retry.Error()
	case errors.Is(err, auth.ErrTooManyAttempts):
		return http.StatusTooManyRequests, auth.ErrTooManyAttempts.Error()
	case errors.Is(err, auth.ErrCodeExpired):
		return http.StatusBadRequest, auth.ErrCodeExpired.Error()
	case errors.Is(err, auth.ErrCodeInvalid):
		return http.StatusBadRequest, auth.ErrCodeInvalid.Error()

	case errors.As(err, &echoE):
		if msg, ok := echoE.Message.(string); ok {
			return echoE.Code, msg
		}
		return echoE.Code, http.StatusText(echoE.Code)
	}
	return http.StatusInternalServerError, internalMessage
}
