package contract

import (
	"encoding/json"
	"fmt"
)

// Analysis는 AI 서버의 분석 결과를 우리 언어로 옮긴 것이다. 금액은 원 단위다.
type Analysis struct {
	RiskScore     int32
	Grade         Severity
	LossAmount    int64
	LossMin       int64
	LossMax       int64
	Deposit       *int64
	Clauses       []AnalysisClause
	Flow          json.RawMessage
	Warnings      []string
	ModelVersion  string
	PolicyVersion string
}

type AnalysisClause struct {
	Original     string
	ClauseType   string
	Confidence   float64
	Severity     Severity
	LMax         int64
	ExpectedLoss int64
	Reason       string
	LawReference []string
	SafeText     string
}

// Result는 분석이 끝났을 때 계약서에 채워 넣는 값이다.
// 파일 이름 등 업로드 시점에 정해진 값은 포함하지 않는다.
type Result struct {
	LossAmount    int32
	RiskScore     int32
	Grade         Severity
	LossMin       int32
	LossMax       int32
	DepositAmount *int32
	ModelVersion  string
	Warnings      []string
	RiskFlow      json.RawMessage
}

// ResultFrom은 분석 결과에서 계약서에 채울 값을 만든다.
func ResultFrom(a Analysis) Result {
	warnings := a.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return Result{
		LossAmount:    ClampInt32(a.LossAmount),
		RiskScore:     a.RiskScore,
		Grade:         a.Grade,
		LossMin:       ClampInt32(a.LossMin),
		LossMax:       ClampInt32(a.LossMax),
		DepositAmount: ClampOptional(a.Deposit),
		ModelVersion:  fmt.Sprintf("%s / %s", a.ModelVersion, a.PolicyVersion),
		Warnings:      warnings,
		RiskFlow:      a.Flow,
	}
}

// ClausesFrom은 분석 결과의 위험 조항을 저장할 형태로 바꾼다. ID는 호출하는 쪽에서 채운다.
func ClausesFrom(contractID string, a Analysis) []Clause {
	out := make([]Clause, len(a.Clauses))
	for i, c := range a.Clauses {
		clauseType := c.ClauseType
		confidence := float32(c.Confidence)
		lMax := ClampInt32(c.LMax)
		out[i] = Clause{
			ContractID:   contractID,
			Original:     c.Original,
			ClauseType:   &clauseType,
			Confidence:   &confidence,
			LMax:         &lMax,
			LossAmount:   ClampInt32(c.ExpectedLoss),
			Severity:     c.Severity,
			Reason:       c.Reason,
			LawReference: c.LawReference,
			SafeText:     c.SafeText,
			OrderIndex:   int32(i),
		}
	}
	return out
}
