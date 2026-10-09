// Package contract contains the pure data and rules for contract analysis.
// It imports only the standard library. It knows nothing about the DB, HTTP, or the AI server.
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

// Progress is the processing stage while analysis is pending.
// Stage: reading | ocr | splitting | classifying | scoring
type Progress struct {
	Stage string `json:"stage"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

// InitialProgress is the progress state right after upload.
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

// Clause is a risk clause found in a contract.
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

// Summary is a one-line summary shown in lists.
type Summary struct {
	ID           string
	FileName     string
	CreatedAt    time.Time
	Status       Status
	ErrorMessage *string
	LossAmount   int32
	ClauseCount  int32
}

// Page is one page of history.
type Page struct {
	Items   []Summary
	Page    int
	Size    int
	Total   int
	HasMore bool
}
