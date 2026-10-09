// Package app는 모든 계층을 조립한다(수동 DI). 계층 전부를 알아도 되는 유일한 곳이다.
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
	// 분석 작업의 전체 제한은 AI 서버 대기 시간에 저장 시간 여유를 더한다.
	analysisSaveMargin = time.Minute
)

// Overrides는 테스트에서 외부 연동을 가짜로 바꿀 때 쓴다. 비워 두면 실제 구현을 쓴다.
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

// New는 DB에 연결해 마이그레이션을 적용하고 서버를 조립한다.
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

// Build는 이미 준비된 DB 풀 위에서 서버를 조립한다.
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

// Run은 서버를 띄우고 ctx가 끝나면 진행 중인 요청과 분석을 기다린 뒤 정리한다.
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

// housekeeping은 서버가 내려가며 사라진 분석과 만료된 인증 데이터를 주기적으로 정리한다.
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

// PoolForTest는 통합 테스트에서 상태를 직접 조작할 수 있게 풀을 돌려준다.
func (a *App) PoolForTest() *pgxpool.Pool { return a.pool }
