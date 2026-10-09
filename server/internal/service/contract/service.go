// Package contract assembles the flow of contract upload, background analysis, and lookup/deletion.
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

	// Analysis jobs run detached from the request context. Shutdown waits for them on exit.
	jobTimeout time.Duration
	baseCtx    context.Context
	cancelBase context.CancelFunc
	jobs       sync.WaitGroup
}

func NewService(repo Repository, analyzer Analyzer, clk clock.Clock, log *slog.Logger, jobTimeout time.Duration) *Service {
	base, cancel := context.WithCancel(context.Background())
	return &Service{repo: repo, analyzer: analyzer, clock: clk, log: log, jobTimeout: jobTimeout, baseCtx: base, cancelBase: cancel}
}

// Submit validates the upload, creates a pending contract, and starts the analysis. It does not wait for the analysis.
func (s *Service) Submit(ctx context.Context, userID, fileName, contentType string, size int64, body io.Reader) (string, error) {
	if err := contract.ValidateUpload(fileName, contentType, size); err != nil {
		return "", err
	}
	// the file is used after the request ends, so read it fully into memory
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

// runAnalysis runs the AI analysis and saves the result. On failure it marks the contract failed.
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

// fail uses a separate context so it can record the failure state even after cancellation by the shutdown signal.
func (s *Service) fail(id, message string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(s.baseCtx), 5*time.Second)
	defer cancel()
	if err := s.repo.Fail(ctx, id, message); err != nil {
		s.log.Error("mark contract failed", "contract", id, "error", err)
	}
}

// Get returns the caller's own contract. Clauses are included only when it is completed.
// If pending is stale, it is treated as a lost job and marked failed before being returned.
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

// Clause returns one clause of a completed contract.
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

// List returns one page of history (newest first, including failed analyses).
func (s *Service) List(ctx context.Context, userID string, page, size int) (contract.Page, error) {
	return s.repo.Page(ctx, userID, page, size)
}

// Recent returns up to limit completed and in-progress analyses, newest first (failed excluded).
func (s *Service) Recent(ctx context.Context, userID string, limit int) ([]contract.Summary, error) {
	return s.repo.Recent(ctx, userID, limit)
}

// Delete deletes the caller's contract and its analysis results.
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

// SweepStale marks stale pending contracts as failed. It cleans up jobs lost when the server went down.
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

// Shutdown waits for in-flight analyses until ctx ends. After that, it cancels the remaining jobs.
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
