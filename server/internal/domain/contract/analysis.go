package contract

import (
	"encoding/json"
	"fmt"
)

// Analysis is the AI server's analysis result translated into our own terms. Amounts are in won.
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

// Result holds the values filled into a contract when analysis finishes.
// It does not include values fixed at upload time, such as the file name.
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

// ResultFrom builds the values to fill into a contract from an analysis result.
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

// ClausesFrom converts the risk clauses of an analysis result into a storable form. The caller fills in the IDs.
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
