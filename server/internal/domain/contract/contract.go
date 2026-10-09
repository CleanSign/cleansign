// Package contract는 계약서 분석의 순수한 데이터와 규칙을 담는다.
// 표준 라이브러리만 import한다. DB, HTTP, AI 서버를 알지 못한다.
package contract

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// Progress는 분석 중(pending)일 때의 처리 단계다.
// Stage: reading | ocr | splitting | classifying | scoring
type Progress struct {
	Stage string `json:"stage"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

// InitialProgress는 업로드 직후의 진행 상태다.
func InitialProgress() Progress { return Progress{Stage: "reading"} }

type Contract struct {
	ID            string
	UserID        string
	FileName      string
	MimeType      string
	Status        Status
	LossAmount    int32
	RiskScore     *int32
	Grade         *Severity
	LossMin       *int32
	LossMax       *int32
	DepositAmount *int32
	ModelVersion  *string
	Warnings      []string
	RiskFlow      json.RawMessage
	Progress      *Progress
	ErrorMessage  *string
	CreatedAt     time.Time
}

// Clause는 계약서에서 찾은 위험 조항이다.
type Clause struct {
	ID           string
	ContractID   string
	Original     string
	ClauseType   *string
	Confidence   *float32
	LMax         *int32
	LossAmount   int32
	Severity     Severity
	Reason       string
	LawReference []string
	SafeText     string
	OrderIndex   int32
}

// Summary는 목록에 보여 줄 한 줄 요약이다.
type Summary struct {
	ID           string
	FileName     string
	CreatedAt    time.Time
	Status       Status
	ErrorMessage *string
	LossAmount   int32
	ClauseCount  int32
}

// Page는 히스토리 한 페이지다.
type Page struct {
	Items   []Summary
	Page    int
	Size    int
	Total   int
	HasMore bool
}
