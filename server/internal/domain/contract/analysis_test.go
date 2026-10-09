package contract

import "testing"

func sampleAnalysis() Analysis {
	deposit := int64(300_000_000)
	return Analysis{
		RiskScore:     72,
		Grade:         SeverityHigh,
		LossAmount:    50_000_000,
		LossMin:       20_000_000,
		LossMax:       9_000_000_000, // 상한을 넘는다
		Deposit:       &deposit,
		Warnings:      nil,
		ModelVersion:  "gemma4:12b",
		PolicyVersion: "2026.10",
		Clauses: []AnalysisClause{
			{Original: "a", ClauseType: "선순위근저당", Confidence: 0.9, Severity: SeverityHigh, LMax: 100, ExpectedLoss: 40, Reason: "r1", LawReference: []string{"민법 제1조", "민법 제2조"}, SafeText: "s1"},
			{Original: "b", ClauseType: "기타불이익", Confidence: 0.5, Severity: SeverityLow, LMax: 5_000_000_000, ExpectedLoss: -3, Reason: "r2", SafeText: "s2"},
		},
	}
}

func TestResultFrom(t *testing.T) {
	r := ResultFrom(sampleAnalysis())
	if r.LossAmount != 50_000_000 || r.RiskScore != 72 || r.Grade != SeverityHigh {
		t.Errorf("기본 값이 다르다: %+v", r)
	}
	if r.LossMax != 2_147_483_647 {
		t.Errorf("lossMax는 상한으로 잘려야 한다: %d", r.LossMax)
	}
	if r.DepositAmount == nil || *r.DepositAmount != 300_000_000 {
		t.Errorf("보증금: %v", r.DepositAmount)
	}
	if r.ModelVersion != "gemma4:12b / 2026.10" {
		t.Errorf("modelVersion: %q", r.ModelVersion)
	}
	if r.Warnings == nil || len(r.Warnings) != 0 {
		t.Errorf("warnings는 빈 배열이어야 한다: %#v", r.Warnings)
	}
}

func TestResultFromWithoutDeposit(t *testing.T) {
	a := sampleAnalysis()
	a.Deposit = nil
	if r := ResultFrom(a); r.DepositAmount != nil {
		t.Errorf("보증금이 없으면 nil이어야 한다: %v", *r.DepositAmount)
	}
}

func TestClausesFrom(t *testing.T) {
	cl := ClausesFrom("c-1", sampleAnalysis())
	if len(cl) != 2 {
		t.Fatalf("조항 수: %d", len(cl))
	}
	if cl[0].ContractID != "c-1" || cl[0].OrderIndex != 0 || cl[1].OrderIndex != 1 {
		t.Errorf("순서/소속: %+v %+v", cl[0], cl[1])
	}
	if cl[0].LossAmount != 40 || cl[0].LawReference[1] != "민법 제2조" {
		t.Errorf("첫 조항: %+v", cl[0])
	}
	if cl[1].LossAmount != 0 { // 음수는 0으로
		t.Errorf("음수 손실은 0이어야 한다: %d", cl[1].LossAmount)
	}
	if *cl[1].LMax != 2_147_483_647 {
		t.Errorf("lMax 상한: %d", *cl[1].LMax)
	}
}
