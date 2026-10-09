package contract

import (
	"context"
	"time"

	"github.com/CleanSign/cleansign/server/internal/domain/contract"
)

// Repository is the contract store port. It returns errs.ErrNotFound if the target is missing or not owned by the caller.
type Repository interface {
	Create(ctx context.Context, c contract.Contract) error
	// Get returns the caller's own contract.
	Get(ctx context.Context, userID, id string) (contract.Contract, error)
	Clauses(ctx context.Context, contractID string) ([]contract.Clause, error)
	Page(ctx context.Context, userID string, page, size int) (contract.Page, error)
	// Recent returns only completed and in-progress analyses, newest first (failed excluded).
	Recent(ctx context.Context, userID string, limit int) ([]contract.Summary, error)
	// Delete returns true if it deleted something. Clauses are deleted with it.
	Delete(ctx context.Context, userID, id string) (bool, error)
	UpdateProgress(ctx context.Context, id string, p contract.Progress) error
	// Complete saves the result and clauses in one transaction.
	// If the user deleted the contract during analysis, it saves nothing and returns false.
	Complete(ctx context.Context, id string, r contract.Result, clauses []contract.Clause) (bool, error)
	Fail(ctx context.Context, id, message string) error
	// FailIfStale marks only pending contracts created before createdBefore as failed.
	FailIfStale(ctx context.Context, id string, createdBefore time.Time, message string) error
	// FailAllStale marks all stale pending contracts as failed and returns the number changed.
	FailAllStale(ctx context.Context, createdBefore time.Time, message string) (int64, error)
}

// AnalysisError carries the reason an analysis failed together with a message to show the user.
// Analyzer returns every failure in this form.
type AnalysisError struct {
	Message string
	Err     error
}

func (e *AnalysisError) Error() string { return e.Message }
func (e *AnalysisError) Unwrap() error { return e.Err }

// ProgressFunc is called each time the analysis stage changes. Returning an error stops the analysis.
type ProgressFunc func(contract.Progress) error

// Analyzer is the AI analysis server port.
type Analyzer interface {
	Analyze(ctx context.Context, fileName string, pdf []byte, onProgress ProgressFunc) (contract.Analysis, error)
}
