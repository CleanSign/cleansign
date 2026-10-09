package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/CleanSign/cleansign/server/internal/database/sqlc"
	"github.com/CleanSign/cleansign/server/internal/domain/contract"
)

// lawReferenceSep는 여러 법령을 한 컬럼에 이어 붙일 때 쓰는 구분자다. 기존 데이터 형식과 같다.
const lawReferenceSep = ", "

// ContractRepo는 계약서 포트를 구현한다.
type ContractRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewContractRepo(pool *pgxpool.Pool) *ContractRepo {
	return &ContractRepo{pool: pool, q: queries(pool)}
}

func (r *ContractRepo) Create(ctx context.Context, c contract.Contract) error {
	progress, err := marshalProgress(c.Progress)
	if err != nil {
		return err
	}
	return r.q.CreateContract(ctx, sqlc.CreateContractParams{
		ID: c.ID, UserID: c.UserID, FileName: c.FileName, MimeType: c.MimeType, Progress: progress,
	})
}

func (r *ContractRepo) Get(ctx context.Context, userID, id string) (contract.Contract, error) {
	row, err := r.q.GetContractForUser(ctx, sqlc.GetContractForUserParams{ID: id, UserID: userID})
	if err != nil {
		return contract.Contract{}, notFound(err)
	}
	return toContract(row)
}

func (r *ContractRepo) Clauses(ctx context.Context, contractID string) ([]contract.Clause, error) {
	rows, err := r.q.ListClausesByContract(ctx, contractID)
	if err != nil {
		return nil, err
	}
	out := make([]contract.Clause, len(rows))
	for i, row := range rows {
		out[i] = toClause(row)
	}
	return out, nil
}

func (r *ContractRepo) Page(ctx context.Context, userID string, page, size int) (contract.Page, error) {
	rows, err := r.q.ListContractsPage(ctx, sqlc.ListContractsPageParams{UserID: userID, Limit: int32(size), Offset: int32((page - 1) * size)})
	if err != nil {
		return contract.Page{}, err
	}
	total, err := r.q.CountContracts(ctx, userID)
	if err != nil {
		return contract.Page{}, err
	}
	items := make([]contract.Summary, len(rows))
	for i, row := range rows {
		items[i] = contract.Summary{
			ID: row.ID, FileName: row.FileName, CreatedAt: row.CreatedAt.UTC(), Status: contract.Status(row.Status),
			ErrorMessage: row.ErrorMessage, LossAmount: row.LossAmount, ClauseCount: row.ClauseCount,
		}
	}
	return contract.Page{Items: items, Page: page, Size: size, Total: int(total), HasMore: page*size < int(total)}, nil
}

func (r *ContractRepo) Recent(ctx context.Context, userID string, limit int) ([]contract.Summary, error) {
	rows, err := r.q.ListRecentContracts(ctx, sqlc.ListRecentContractsParams{UserID: userID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := make([]contract.Summary, len(rows))
	for i, row := range rows {
		items[i] = contract.Summary{
			ID: row.ID, FileName: row.FileName, CreatedAt: row.CreatedAt.UTC(), Status: contract.Status(row.Status),
			LossAmount: row.LossAmount, ClauseCount: row.ClauseCount,
		}
	}
	return items, nil
}

func (r *ContractRepo) Delete(ctx context.Context, userID, id string) (bool, error) {
	n, err := r.q.DeleteContractForUser(ctx, sqlc.DeleteContractForUserParams{ID: id, UserID: userID})
	return n > 0, err
}

func (r *ContractRepo) UpdateProgress(ctx context.Context, id string, p contract.Progress) error {
	b, err := marshalProgress(&p)
	if err != nil {
		return err
	}
	return r.q.UpdateContractProgress(ctx, sqlc.UpdateContractProgressParams{ID: id, Progress: b})
}

// Complete는 결과와 조항을 한 트랜잭션에 저장한다. 계약서가 이미 지워졌으면 false를 돌려준다.
func (r *ContractRepo) Complete(ctx context.Context, id string, res contract.Result, clauses []contract.Clause) (bool, error) {
	warnings, err := json.Marshal(res.Warnings)
	if err != nil {
		return false, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // 커밋 뒤의 롤백은 무시된다

	q := r.q.WithTx(tx)
	grade := sqlc.ClauseSeverity(res.Grade)
	score := res.RiskScore
	lossMin, lossMax := res.LossMin, res.LossMax
	model := res.ModelVersion
	updated, err := q.CompleteContract(ctx, sqlc.CompleteContractParams{
		ID: id, LossAmount: res.LossAmount, RiskScore: &score, Grade: &grade,
		LossMin: &lossMin, LossMax: &lossMax, DepositAmount: res.DepositAmount,
		ModelVersion: &model, Warnings: warnings, RiskFlow: nilIfJSONNull(res.RiskFlow),
	})
	if err != nil {
		return false, err
	}
	if updated == 0 {
		return false, nil // 분석하는 동안 사용자가 지웠다: 결과를 버린다
	}
	for _, c := range clauses {
		if err := q.InsertClause(ctx, insertClauseParams(c)); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (r *ContractRepo) Fail(ctx context.Context, id, message string) error {
	_, err := r.q.FailContract(ctx, sqlc.FailContractParams{ID: id, ErrorMessage: &message})
	return err
}

func (r *ContractRepo) FailIfStale(ctx context.Context, id string, createdBefore time.Time, message string) error {
	_, err := r.q.FailStaleContract(ctx, sqlc.FailStaleContractParams{ID: id, CreatedAt: createdBefore, ErrorMessage: &message})
	return err
}

func (r *ContractRepo) FailAllStale(ctx context.Context, createdBefore time.Time, message string) (int64, error) {
	return r.q.FailStaleContracts(ctx, sqlc.FailStaleContractsParams{CreatedAt: createdBefore, ErrorMessage: &message})
}

// --- 변환 ---

func toContract(r sqlc.Contract) (contract.Contract, error) {
	c := contract.Contract{
		ID: r.ID, UserID: r.UserID, FileName: r.FileName, MimeType: r.MimeType,
		Status: contract.Status(r.Status), LossAmount: r.LossAmount, RiskScore: r.RiskScore,
		LossMin: r.LossMin, LossMax: r.LossMax, DepositAmount: r.DepositAmount, ModelVersion: r.ModelVersion,
		ErrorMessage: r.ErrorMessage, CreatedAt: r.CreatedAt.UTC(),
	}
	if r.Grade != nil {
		g := contract.Severity(*r.Grade)
		c.Grade = &g
	}
	if len(r.Warnings) > 0 {
		if err := json.Unmarshal(r.Warnings, &c.Warnings); err != nil {
			return contract.Contract{}, fmt.Errorf("decode warnings: %w", err)
		}
	}
	c.RiskFlow = nilIfJSONNull(r.RiskFlow)
	if len(r.Progress) > 0 {
		var p *contract.Progress
		if err := json.Unmarshal(r.Progress, &p); err != nil {
			return contract.Contract{}, fmt.Errorf("decode progress: %w", err)
		}
		c.Progress = p
	}
	return c, nil
}

func toClause(r sqlc.ContractDangerClause) contract.Clause {
	c := contract.Clause{
		ID: r.ID, ContractID: r.ContractID, Original: r.Original, ClauseType: r.ClauseType,
		Confidence: r.Confidence, LMax: r.LMax, LossAmount: r.LossAmount, Severity: contract.Severity(r.Severity),
		Reason: r.Reason, SafeText: r.SafeText, OrderIndex: r.OrderIndex,
	}
	if r.LawReference != nil && *r.LawReference != "" {
		c.LawReference = strings.Split(*r.LawReference, lawReferenceSep)
	}
	return c
}

func insertClauseParams(c contract.Clause) sqlc.InsertClauseParams {
	var law *string
	if joined := strings.Join(c.LawReference, lawReferenceSep); joined != "" {
		law = &joined
	}
	return sqlc.InsertClauseParams{
		ID: c.ID, ContractID: c.ContractID, Original: c.Original, ClauseType: c.ClauseType,
		Confidence: c.Confidence, LMax: c.LMax, LossAmount: c.LossAmount, Severity: sqlc.ClauseSeverity(c.Severity),
		Reason: c.Reason, LawReference: law, SafeText: c.SafeText, OrderIndex: c.OrderIndex,
	}
}

func marshalProgress(p *contract.Progress) ([]byte, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(p)
}

// nilIfJSONNull은 비어 있거나 JSON null인 값을 nil로 돌려준다(컬럼은 NULL로 저장한다).
func nilIfJSONNull(b json.RawMessage) json.RawMessage {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return b
}
