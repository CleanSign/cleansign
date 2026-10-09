package contract

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CleanSign/cleansign/server/internal/domain/contract"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
	"github.com/CleanSign/cleansign/server/internal/shared/errs"
)

// fakeRepo is a Repository that stores in memory.
type fakeRepo struct {
	mu        sync.Mutex
	contracts map[string]contract.Contract
	clauses   map[string][]contract.Clause
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{contracts: map[string]contract.Contract{}, clauses: map[string][]contract.Clause{}}
}

func (r *fakeRepo) Create(_ context.Context, c contract.Contract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contracts[c.ID] = c
	return nil
}

func (r *fakeRepo) Get(_ context.Context, userID, id string) (contract.Contract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contracts[id]
	if !ok || c.UserID != userID {
		return contract.Contract{}, errs.ErrNotFound
	}
	return c, nil
}

func (r *fakeRepo) Clauses(_ context.Context, id string) ([]contract.Clause, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.clauses[id], nil
}

func (r *fakeRepo) Page(context.Context, string, int, int) (contract.Page, error) {
	return contract.Page{}, nil
}
func (r *fakeRepo) Recent(context.Context, string, int) ([]contract.Summary, error) { return nil, nil }

func (r *fakeRepo) Delete(_ context.Context, userID, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contracts[id]
	if !ok || c.UserID != userID {
		return false, nil
	}
	delete(r.contracts, id)
	delete(r.clauses, id)
	return true, nil
}

func (r *fakeRepo) UpdateProgress(_ context.Context, id string, p contract.Progress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.contracts[id]; ok {
		c.Progress = &p
		r.contracts[id] = c
	}
	return nil
}

func (r *fakeRepo) Complete(_ context.Context, id string, res contract.Result, cl []contract.Clause) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contracts[id]
	if !ok {
		return false, nil
	}
	c.Status = contract.StatusCompleted
	c.LossAmount = res.LossAmount
	c.Progress = nil
	r.contracts[id] = c
	r.clauses[id] = cl
	return true, nil
}

func (r *fakeRepo) Fail(_ context.Context, id, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.contracts[id]; ok {
		c.Status = contract.StatusFailed
		c.ErrorMessage = &message
		c.Progress = nil
		r.contracts[id] = c
	}
	return nil
}

func (r *fakeRepo) FailIfStale(_ context.Context, id string, before time.Time, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.contracts[id]; ok && c.Status == contract.StatusPending && c.CreatedAt.Before(before) {
		c.Status = contract.StatusFailed
		c.ErrorMessage = &message
		r.contracts[id] = c
	}
	return nil
}

func (r *fakeRepo) FailAllStale(_ context.Context, before time.Time, message string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for id, c := range r.contracts {
		if c.Status == contract.StatusPending && c.CreatedAt.Before(before) {
			c.Status = contract.StatusFailed
			c.ErrorMessage = &message
			r.contracts[id] = c
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) status(id string) (contract.Contract, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contracts[id]
	return c, ok
}

// fakeAnalyzer is an Analyzer that returns a preset result or error.
type fakeAnalyzer struct {
	result contract.Analysis
	err    error
	gate   chan struct{} // if set, waits until it is opened
	panics bool
}

func (a *fakeAnalyzer) Analyze(ctx context.Context, _ string, _ []byte, onProgress ProgressFunc) (contract.Analysis, error) {
	if err := onProgress(contract.Progress{Stage: "classifying", Done: 1, Total: 2}); err != nil {
		return contract.Analysis{}, err
	}
	if a.panics {
		panic("boom")
	}
	if a.gate != nil {
		select {
		case <-a.gate:
		case <-ctx.Done():
			return contract.Analysis{}, ctx.Err()
		}
	}
	return a.result, a.err
}

func newService(repo *fakeRepo, an *fakeAnalyzer) (*Service, *clock.Fixed) {
	clk := &clock.Fixed{T: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(repo, an, clk, log, time.Minute), clk
}

func sampleResult() contract.Analysis {
	return contract.Analysis{
		RiskScore: 70, Grade: contract.SeverityHigh, LossAmount: 1000, LossMin: 500, LossMax: 1500,
		ModelVersion: "m", PolicyVersion: "p",
		Clauses: []contract.AnalysisClause{
			{Original: "a", ClauseType: "t", Severity: contract.SeverityHigh, ExpectedLoss: 400, Reason: "r", SafeText: "s"},
			{Original: "b", ClauseType: "t", Severity: contract.SeverityLow, ExpectedLoss: 100, Reason: "r", SafeText: "s"},
		},
	}
}

func submit(t *testing.T, s *Service, user string) string {
	t.Helper()
	id, err := s.Submit(context.Background(), user, "a.pdf", "application/pdf", 4, strings.NewReader("%PDF"))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return id
}

func waitDone(t *testing.T, s *Service) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Shutdown(ctx)
}

func TestSubmitValidatesUpload(t *testing.T) {
	s, _ := newService(newFakeRepo(), &fakeAnalyzer{})
	ctx := context.Background()
	if _, err := s.Submit(ctx, "u1", "a.txt", "text/plain", 4, strings.NewReader("x")); !errors.Is(err, contract.ErrNotPDF) {
		t.Errorf("PDF가 아니면 ErrNotPDF: %v", err)
	}
	if _, err := s.Submit(ctx, "u1", "a.pdf", "application/pdf", 0, strings.NewReader("")); !errors.Is(err, contract.ErrBadSize) {
		t.Errorf("빈 파일은 ErrBadSize: %v", err)
	}
	big := bytes.NewReader(make([]byte, contract.MaxUploadBytes+1))
	if _, err := s.Submit(ctx, "u1", "a.pdf", "application/pdf", 1, big); !errors.Is(err, contract.ErrBadSize) {
		t.Errorf("선언한 크기와 달리 실제로 크면 ErrBadSize: %v", err)
	}
}

func TestSubmitCompletesInBackground(t *testing.T) {
	repo := newFakeRepo()
	s, _ := newService(repo, &fakeAnalyzer{result: sampleResult()})
	id := submit(t, s, "u1")

	// right after the response the state is pending and has progress (the analysis keeps running)
	if c, ok := repo.status(id); !ok || (c.Status != contract.StatusPending && c.Status != contract.StatusCompleted) {
		t.Fatalf("제출 직후 상태: %+v", c)
	}
	waitDone(t, s)

	c, clauses, err := s.Get(context.Background(), "u1", id)
	if err != nil || c.Status != contract.StatusCompleted {
		t.Fatalf("완료돼야 한다: %+v, %v", c, err)
	}
	if len(clauses) != 2 || clauses[0].ID == "" || clauses[0].ID == clauses[1].ID {
		t.Errorf("조항에 서로 다른 ID가 있어야 한다: %+v", clauses)
	}
}

func TestAnalysisFailureMessages(t *testing.T) {
	cases := []struct {
		name string
		an   *fakeAnalyzer
		want string
	}{
		{"사용자에게 보여 줄 실패", &fakeAnalyzer{err: &AnalysisError{Message: "글자를 읽지 못했어요"}}, "글자를 읽지 못했어요"},
		{"알 수 없는 오류", &fakeAnalyzer{err: errors.New("db down")}, saveFailedMsg},
		{"panic", &fakeAnalyzer{panics: true}, saveFailedMsg},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := newFakeRepo()
			s, _ := newService(repo, c.an)
			id := submit(t, s, "u1")
			waitDone(t, s)
			got, _ := repo.status(id)
			if got.Status != contract.StatusFailed || got.ErrorMessage == nil || *got.ErrorMessage != c.want {
				t.Errorf("failed/%q이어야 한다: %+v", c.want, got)
			}
			if got.Progress != nil {
				t.Error("실패하면 진행 상황을 지운다")
			}
		})
	}
}

func TestDeletedDuringAnalysisDiscardsResult(t *testing.T) {
	repo := newFakeRepo()
	gate := make(chan struct{})
	s, _ := newService(repo, &fakeAnalyzer{result: sampleResult(), gate: gate})
	id := submit(t, s, "u1")

	if err := s.Delete(context.Background(), "u1", id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	close(gate)
	waitDone(t, s)

	if _, ok := repo.status(id); ok {
		t.Error("지운 계약서가 되살아나면 안 된다")
	}
	if len(repo.clauses[id]) != 0 {
		t.Error("지운 계약서의 조항을 저장하면 안 된다")
	}
}

func TestGetOnlyOwnContract(t *testing.T) {
	repo := newFakeRepo()
	s, _ := newService(repo, &fakeAnalyzer{result: sampleResult()})
	id := submit(t, s, "u1")
	waitDone(t, s)

	if _, _, err := s.Get(context.Background(), "u2", id); !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("남의 계약서는 NotFound: %v", err)
	}
	if err := s.Delete(context.Background(), "u2", id); !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("남의 계약서 삭제는 NotFound: %v", err)
	}
}

func TestGetMarksStalePendingAsFailed(t *testing.T) {
	repo := newFakeRepo()
	gate := make(chan struct{}) // a situation where the analysis never finishes
	s, clk := newService(repo, &fakeAnalyzer{gate: gate})
	id := submit(t, s, "u1")

	clk.Advance(contract.StaleAfter - time.Minute)
	if c, _, _ := s.Get(context.Background(), "u1", id); c.Status != contract.StatusPending {
		t.Errorf("15분 전에는 pending: %v", c.Status)
	}
	clk.Advance(2 * time.Minute)
	c, clauses, err := s.Get(context.Background(), "u1", id)
	if err != nil || c.Status != contract.StatusFailed || c.ErrorMessage == nil || *c.ErrorMessage != contract.StaleMessage {
		t.Errorf("15분이 지나면 failed: %+v %v", c, err)
	}
	if clauses != nil {
		t.Error("완료되지 않았으면 조항이 없다")
	}
	close(gate)
	waitDone(t, s)
}

func TestClause(t *testing.T) {
	repo := newFakeRepo()
	s, _ := newService(repo, &fakeAnalyzer{result: sampleResult()})
	id := submit(t, s, "u1")
	waitDone(t, s)

	_, clauses, _ := s.Get(context.Background(), "u1", id)
	got, err := s.Clause(context.Background(), "u1", id, clauses[1].ID)
	if err != nil || got.Original != "b" {
		t.Errorf("조항을 찾아야 한다: %+v %v", got, err)
	}
	if _, err := s.Clause(context.Background(), "u1", id, "nope"); !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("없는 조항은 NotFound: %v", err)
	}
	if _, err := s.Clause(context.Background(), "u2", id, clauses[0].ID); !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("남의 계약서의 조항은 NotFound: %v", err)
	}
}

func TestClauseRequiresCompleted(t *testing.T) {
	repo := newFakeRepo()
	s, _ := newService(repo, &fakeAnalyzer{err: &AnalysisError{Message: "x"}})
	id := submit(t, s, "u1")
	waitDone(t, s)
	if _, err := s.Clause(context.Background(), "u1", id, "any"); !errors.Is(err, errs.ErrNotFound) {
		t.Errorf("완료되지 않은 계약서의 조항은 NotFound: %v", err)
	}
}

func TestSweepStale(t *testing.T) {
	repo := newFakeRepo()
	gate := make(chan struct{})
	s, clk := newService(repo, &fakeAnalyzer{gate: gate})
	id := submit(t, s, "u1")
	clk.Advance(contract.StaleAfter + time.Minute)

	if err := s.SweepStale(context.Background()); err != nil {
		t.Fatal(err)
	}
	if c, _ := repo.status(id); c.Status != contract.StatusFailed {
		t.Errorf("오래된 pending은 failed가 된다: %v", c.Status)
	}
	close(gate)
	waitDone(t, s)
}

func TestShutdownWaitsForRunningJobs(t *testing.T) {
	repo := newFakeRepo()
	gate := make(chan struct{})
	s, _ := newService(repo, &fakeAnalyzer{result: sampleResult(), gate: gate})
	id := submit(t, s, "u1")

	go func() {
		time.Sleep(50 * time.Millisecond)
		close(gate)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Shutdown(ctx)

	if c, _ := repo.status(id); c.Status != contract.StatusCompleted {
		t.Errorf("종료 전에 진행 중이던 분석이 끝나야 한다: %v", c.Status)
	}
}

func TestShutdownCancelsAfterDeadline(t *testing.T) {
	repo := newFakeRepo()
	s, _ := newService(repo, &fakeAnalyzer{gate: make(chan struct{})}) // an analysis that never finishes
	id := submit(t, s, "u1")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	s.Shutdown(ctx)

	if c, _ := repo.status(id); c.Status != contract.StatusFailed {
		t.Errorf("시간이 지나 취소되면 failed로 남긴다: %v", c.Status)
	}
}
