// Package server holds the HTTP handlers (inbound adapter).
// It only reads requests, calls services, and shapes responses. The rules live in domain and service.
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

// Deps are the services and settings the handlers call.
type Deps struct {
	Auth         *authsvc.Service
	Users        *usersvc.Service
	Contracts    *contractsvc.Service
	Log          *slog.Logger
	CookieSecure bool
	// Health checks the dependent service (DB) at /healthz.
	Health func(ctx context.Context) error
}

type handler struct{ Deps }

var _ api.ServerInterface = (*handler)(nil)

// New creates an Echo with routes and middleware configured.
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

// accessLog writes one line per request. The query string may contain personal data, so it is not logged.
func (h *handler) accessLog(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		if err != nil {
			c.Error(err) // record after the status code is finalized
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
