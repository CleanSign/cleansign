package contract

import (
	"context"
	"time"

	"github.com/CleanSign/cleansign/server/internal/domain/contract"
)

// Repository는 계약서 저장소 포트다. 대상이 없거나 본인 소유가 아니면 errs.ErrNotFound를 돌려준다.
type Repository interface {
	Create(ctx context.Context, c contract.Contract) error
	// Get은 본인 소유의 계약서를 돌려준다.
	Get(ctx context.Context, userID, id string) (contract.Contract, error)
	Clauses(ctx context.Context, contractID string) ([]contract.Clause, error)
	Page(ctx context.Context, userID string, page, size int) (contract.Page, error)
	// Recent는 완료·진행 중인 분석만 최신순으로 돌려준다(실패 제외).
	Recent(ctx context.Context, userID string, limit int) ([]contract.Summary, error)
	// Delete는 지웠으면 true를 돌려준다. 조항은 함께 지워진다.
	Delete(ctx context.Context, userID, id string) (bool, error)
	UpdateProgress(ctx context.Context, id string, p contract.Progress) error
	// Complete는 결과와 조항을 한 트랜잭션에 저장한다.
	// 분석하는 동안 사용자가 지운 경우 아무것도 저장하지 않고 false를 돌려준다.
	Complete(ctx context.Context, id string, r contract.Result, clauses []contract.Clause) (bool, error)
	Fail(ctx context.Context, id, message string) error
	// FailIfStale은 createdBefore 이전에 만든 pending 계약서만 failed로 바꾼다.
	FailIfStale(ctx context.Context, id string, createdBefore time.Time, message string) error
	// FailAllStale은 오래된 pending을 모두 failed로 바꾸고 바꾼 수를 돌려준다.
	FailAllStale(ctx context.Context, createdBefore time.Time, message string) (int64, error)
}

// AnalysisError는 분석에 실패한 이유를 사용자에게 보여 줄 메시지와 함께 담는다.
// Analyzer는 모든 실패를 이 형태로 돌려준다.
type AnalysisError struct {
	Message string
	Err     error
}

func (e *AnalysisError) Error() string { return e.Message }
func (e *AnalysisError) Unwrap() error { return e.Err }

// ProgressFunc는 분석 단계가 바뀔 때마다 불린다. 오류를 돌려주면 분석을 멈춘다.
type ProgressFunc func(contract.Progress) error

// Analyzer는 AI 분석 서버 포트다.
type Analyzer interface {
	Analyze(ctx context.Context, fileName string, pdf []byte, onProgress ProgressFunc) (contract.Analysis, error)
}
