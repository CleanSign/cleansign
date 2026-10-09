// Package contract는 계약서 업로드, 백그라운드 분석, 조회·삭제의 흐름을 조립한다.
package contract

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/CleanSign/cleansign/server/internal/domain/contract"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
	"github.com/CleanSign/cleansign/server/internal/shared/errs"
	"github.com/CleanSign/cleansign/server/internal/shared/ids"
)

const (
	pdfMime       = "application/pdf"
	saveFailedMsg = "분석을 저장하는 중 문제가 생겼어요"
)

type Service struct {
	repo     Repository
	analyzer Analyzer
	clock    clock.Clock
	log      *slog.Logger

	// 분석 작업은 요청 컨텍스트와 분리해 돌린다. 종료할 때 Shutdown이 기다린다.
	jobTimeout time.Duration
	baseCtx    context.Context
	cancelBase context.CancelFunc
	jobs       sync.WaitGroup
}

func NewService(repo Repository, analyzer Analyzer, clk clock.Clock, log *slog.Logger, jobTimeout time.Duration) *Service {
	base, cancel := context.WithCancel(context.Background())
	return &Service{repo: repo, analyzer: analyzer, clock: clk, log: log, jobTimeout: jobTimeout, baseCtx: base, cancelBase: cancel}
}

// Submit은 업로드를 검증하고 pending 계약서를 만든 뒤 분석을 시작한다. 분석은 기다리지 않는다.
func (s *Service) Submit(ctx context.Context, userID, fileName, contentType string, size int64, body io.Reader) (string, error) {
	if err := contract.ValidateUpload(fileName, contentType, size); err != nil {
		return "", err
	}
	// 요청이 끝난 뒤에도 파일을 쓰므로 메모리로 확실히 읽어 둔다
	data, err := io.ReadAll(io.LimitReader(body, contract.MaxUploadBytes+1))
	if err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}
	if err := contract.ValidateUpload(fileName, contentType, int64(len(data))); err != nil {
		return "", err
	}

	progress := contract.InitialProgress()
	c := contract.Contract{
		ID:        ids.New(),
		UserID:    userID,
		FileName:  fileName,
		MimeType:  pdfMime,
		Status:    contract.StatusPending,
		Progress:  &progress,
		CreatedAt: s.clock.Now(),
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return "", err
	}

	s.jobs.Add(1)
	go func() {
		defer s.jobs.Done()
		s.runAnalysis(c.ID, fileName, data)
	}()
	return c.ID, nil
}

// runAnalysis는 AI 분석을 돌리고 결과를 저장한다. 실패하면 계약서를 failed로 바꾼다.
func (s *Service) runAnalysis(id, fileName string, data []byte) {
	ctx, cancel := context.WithTimeout(s.baseCtx, s.jobTimeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			s.log.Error("analysis panicked", "contract", id, "panic", r)
			s.fail(id, saveFailedMsg)
		}
	}()

	analysis, err := s.analyzer.Analyze(ctx, fileName, data, func(p contract.Progress) error {
		return s.repo.UpdateProgress(ctx, id, p)
	})
	if err != nil {
		var ae *AnalysisError
		if errors.As(err, &ae) {
			s.log.Warn("analysis failed", "contract", id, "reason", ae.Message, "error", ae.Err)
			s.fail(id, ae.Message)
			return
		}
		s.log.Error("analysis failed", "contract", id, "error", err)
		s.fail(id, saveFailedMsg)
		return
	}

	clauses := contract.ClausesFrom(id, analysis)
	for i := range clauses {
		clauses[i].ID = ids.New()
	}
	saved, err := s.repo.Complete(ctx, id, contract.ResultFrom(analysis), clauses)
	if err != nil {
		s.log.Error("save analysis failed", "contract", id, "error", err)
		s.fail(id, saveFailedMsg)
		return
	}
	if !saved {
		s.log.Info("contract deleted during analysis; result discarded", "contract", id)
	}
}

// fail은 종료 신호로 취소된 뒤에도 실패 상태를 남길 수 있게 별도 컨텍스트를 쓴다.
func (s *Service) fail(id, message string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(s.baseCtx), 5*time.Second)
	defer cancel()
	if err := s.repo.Fail(ctx, id, message); err != nil {
		s.log.Error("mark contract failed", "contract", id, "error", err)
	}
}

// Get은 본인 소유의 계약서를 돌려준다. 완료된 경우에만 조항을 함께 돌려준다.
// pending이 오래됐으면 사라진 작업으로 보고 failed로 바꾼 뒤 돌려준다.
func (s *Service) Get(ctx context.Context, userID, id string) (contract.Contract, []contract.Clause, error) {
	c, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return contract.Contract{}, nil, err
	}
	now := s.clock.Now()
	if c.IsStale(now) {
		if err := s.repo.FailIfStale(ctx, id, now.Add(-contract.StaleAfter), contract.StaleMessage); err != nil {
			return contract.Contract{}, nil, err
		}
		if c, err = s.repo.Get(ctx, userID, id); err != nil {
			return contract.Contract{}, nil, err
		}
	}
	if c.Status != contract.StatusCompleted {
		return c, nil, nil
	}
	clauses, err := s.repo.Clauses(ctx, id)
	if err != nil {
		return contract.Contract{}, nil, err
	}
	return c, clauses, nil
}

// Clause는 완료된 계약서의 조항 하나를 돌려준다.
func (s *Service) Clause(ctx context.Context, userID, contractID, clauseID string) (contract.Clause, error) {
	c, clauses, err := s.Get(ctx, userID, contractID)
	if err != nil {
		return contract.Clause{}, err
	}
	if c.Status != contract.StatusCompleted {
		return contract.Clause{}, errs.ErrNotFound
	}
	for _, cl := range clauses {
		if cl.ID == clauseID {
			return cl, nil
		}
	}
	return contract.Clause{}, errs.ErrNotFound
}

// List는 히스토리 한 페이지(최신순, 실패한 분석 포함)를 돌려준다.
func (s *Service) List(ctx context.Context, userID string, page, size int) (contract.Page, error) {
	return s.repo.Page(ctx, userID, page, size)
}

// Recent는 완료·진행 중인 분석을 최신순으로 limit개 돌려준다(실패 제외).
func (s *Service) Recent(ctx context.Context, userID string, limit int) ([]contract.Summary, error) {
	return s.repo.Recent(ctx, userID, limit)
}

// Delete는 본인 계약서와 분석 결과를 지운다.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return err
	}
	if !deleted {
		return errs.ErrNotFound
	}
	return nil
}

// SweepStale은 오래된 pending을 failed로 바꾼다. 서버가 내려가며 사라진 작업을 정리한다.
func (s *Service) SweepStale(ctx context.Context) error {
	n, err := s.repo.FailAllStale(ctx, s.clock.Now().Add(-contract.StaleAfter), contract.StaleMessage)
	if err != nil {
		return err
	}
	if n > 0 {
		s.log.Info("stale analyses marked failed", "count", n)
	}
	return nil
}

// Shutdown은 진행 중인 분석을 ctx가 끝날 때까지 기다린다. 시간이 지나면 남은 작업을 취소한다.
func (s *Service) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.jobs.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		s.log.Warn("shutdown timeout; cancelling running analyses")
		s.cancelBase()
		<-done
	}
	s.cancelBase()
}
