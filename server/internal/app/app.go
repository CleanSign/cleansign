// Package app wires all layers together (manual DI). It is the only place allowed to know every layer.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/CleanSign/cleansign/server/internal/config"
	"github.com/CleanSign/cleansign/server/internal/database"
	"github.com/CleanSign/cleansign/server/internal/infra/ai"
	"github.com/CleanSign/cleansign/server/internal/infra/mail"
	"github.com/CleanSign/cleansign/server/internal/infra/token"
	"github.com/CleanSign/cleansign/server/internal/server"
	authsvc "github.com/CleanSign/cleansign/server/internal/service/auth"
	contractsvc "github.com/CleanSign/cleansign/server/internal/service/contract"
	usersvc "github.com/CleanSign/cleansign/server/internal/service/user"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
)

const (
	housekeepingInterval = time.Minute
	httpShutdownGrace    = 10 * time.Second
	analysisShutdownWait = 30 * time.Second
	// The overall limit for an analysis job is the AI server wait time plus slack for saving.
	analysisSaveMargin = time.Minute
)

// Overrides replaces external integrations with fakes in tests. Leave empty to use the real implementations.
type Overrides struct {
	Analyzer contractsvc.Analyzer
	Mailer   authsvc.Mailer
	Clock    clock.Clock
}

type App struct {
	Echo      *echo.Echo
	Contracts *contractsvc.Service
	Auth      *authsvc.Service

	pool *pgxpool.Pool
	log  *slog.Logger
}

// New connects to the DB, applies migrations, and assembles the server.
func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*App, error) {
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	log.Info("database ready")
	return Build(pool, cfg, log, Overrides{}), nil
}

// Build assembles the server on top of an already prepared DB pool.
func Build(pool *pgxpool.Pool, cfg config.Config, log *slog.Logger, o Overrides) *App {
	clk := o.Clock
	if clk == nil {
		clk = clock.Real{}
	}
	analyzer := o.Analyzer
	if analyzer == nil {
		analyzer = ai.New(cfg.AI.URL, cfg.AI.Token, cfg.AI.Timeout)
	}
	mailer := o.Mailer
	if mailer == nil {
		sender := mail.New(mail.Config{
			Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, User: cfg.SMTP.User, Password: cfg.SMTP.Password, From: cfg.SMTP.From,
		}, log)
		if !sender.Configured() {
			log.Warn("SMTP가 설정되지 않아 인증코드를 메일 대신 로그에 남깁니다 (개발용)")
		}
		mailer = sender
	}

	users := database.NewUserRepo(pool)
	authService := authsvc.NewService(
		database.NewCodeRepo(pool), users, database.NewSessionRepo(pool),
		mailer, token.New(cfg.JWTSecret), clk, cfg.JWTSecret, log,
	)
	userService := usersvc.NewService(users, clk)
	contractService := contractsvc.NewService(database.NewContractRepo(pool), analyzer, clk, log, cfg.AI.Timeout+analysisSaveMargin)

	e := server.New(server.Deps{
		Auth: authService, Users: userService, Contracts: contractService, Log: log,
		CookieSecure: cfg.CookieSecure, Health: pool.Ping,
	})
	return &App{Echo: e, Contracts: contractService, Auth: authService, pool: pool, log: log}
}

// Run starts the server and, once ctx ends, waits for in-flight requests and analyses before cleaning up.
func (a *App) Run(ctx context.Context, port string) error {
	housekeepingCtx, stopHousekeeping := context.WithCancel(context.Background())
	housekeepingDone := make(chan struct{})
	go func() {
		defer close(housekeepingDone)
		a.housekeeping(housekeepingCtx)
	}()

	serveErr := make(chan error, 1)
	go func() {
		a.log.Info("server listening", "port", port)
		if err := a.Echo.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	var runErr error
	select {
	case runErr = <-serveErr:
	case <-ctx.Done():
		a.log.Info("shutting down")
	}

	httpCtx, cancelHTTP := context.WithTimeout(context.Background(), httpShutdownGrace)
	defer cancelHTTP()
	if err := a.Echo.Shutdown(httpCtx); err != nil {
		a.log.Warn("http shutdown", "error", err)
	}
	jobsCtx, cancelJobs := context.WithTimeout(context.Background(), analysisShutdownWait)
	defer cancelJobs()
	a.Contracts.Shutdown(jobsCtx)

	stopHousekeeping()
	<-housekeepingDone
	a.pool.Close()
	return runErr
}

// housekeeping periodically cleans up analyses lost when the server went down and expired auth data.
func (a *App) housekeeping(ctx context.Context) {
	run := func() {
		if err := a.Contracts.SweepStale(ctx); err != nil && ctx.Err() == nil {
			a.log.Error("sweep stale analyses", "error", err)
		}
		if err := a.Auth.PurgeExpired(ctx); err != nil && ctx.Err() == nil {
			a.log.Error("purge expired auth data", "error", err)
		}
	}
	run()
	t := time.NewTicker(housekeepingInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

// PoolForTest returns the pool so integration tests can manipulate state directly.
func (a *App) PoolForTest() *pgxpool.Pool { return a.pool }
