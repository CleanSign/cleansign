// Package server는 HTTP handler(들어오는 쪽 어댑터)를 담는다.
// 요청을 읽어 service를 부르고 응답 모양으로 바꾸기만 한다. 규칙은 domain과 service에 있다.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/CleanSign/cleansign/server/api"
	authsvc "github.com/CleanSign/cleansign/server/internal/service/auth"
	contractsvc "github.com/CleanSign/cleansign/server/internal/service/contract"
	usersvc "github.com/CleanSign/cleansign/server/internal/service/user"
)

const apiBase = "/api/v1"

// Deps는 handler가 부르는 service와 설정이다.
type Deps struct {
	Auth         *authsvc.Service
	Users        *usersvc.Service
	Contracts    *contractsvc.Service
	Log          *slog.Logger
	CookieSecure bool
	// Health는 /healthz에서 의존 서비스(DB)를 확인한다.
	Health func(ctx context.Context) error
}

type handler struct{ Deps }

var _ api.ServerInterface = (*handler)(nil)

// New는 라우트와 미들웨어가 설정된 Echo를 만든다.
func New(d Deps) *echo.Echo {
	h := &handler{d}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = h.errorHandler

	e.Use(middleware.RequestID())
	e.Use(h.accessLog)
	e.Use(middleware.Recover())
	e.Use(h.authenticate)

	e.GET("/healthz", h.healthz)
	api.RegisterHandlersWithBaseURL(e, h, apiBase)
	return e
}

func (h *handler) healthz(c echo.Context) error {
	if h.Health != nil {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
		defer cancel()
		if err := h.Health(ctx); err != nil {
			h.Log.Error("health check failed", "error", err)
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		}
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// accessLog는 요청마다 한 줄을 남긴다. 쿼리 문자열은 개인정보가 들어갈 수 있어 남기지 않는다.
func (h *handler) accessLog(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		if err != nil {
			c.Error(err) // 상태 코드를 확정한 뒤에 기록한다
		}
		h.Log.Info("request",
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"status", c.Response().Status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
		)
		return nil
	}
}
