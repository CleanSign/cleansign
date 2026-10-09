package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CleanSign/cleansign/server/internal/app"
	"github.com/CleanSign/cleansign/server/internal/config"
	"github.com/CleanSign/cleansign/server/internal/domain/contract"
	contractsvc "github.com/CleanSign/cleansign/server/internal/service/contract"
	"github.com/CleanSign/cleansign/server/internal/testutil/pgtest"
)

// --- 가짜 외부 연동 ---

type fakeMailer struct {
	mu   sync.Mutex
	code map[string]string
}

func (m *fakeMailer) SendCode(_ context.Context, to, code string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.code[to] = code
	return nil
}

// last는 서버가 정규화(소문자)한 주소로 보낸 마지막 코드를 돌려준다.
func (m *fakeMailer) last(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.code[strings.ToLower(to)]
}

type fakeAnalyzer struct {
	mu     sync.Mutex
	result contract.Analysis
	err    error
	gate   chan struct{}
}

func (a *fakeAnalyzer) Analyze(ctx context.Context, _ string, _ []byte, onProgress contractsvc.ProgressFunc) (contract.Analysis, error) {
	if err := onProgress(contract.Progress{Stage: "classifying", Done: 1, Total: 3}); err != nil {
		return contract.Analysis{}, err
	}
	a.mu.Lock()
	gate, res, err := a.gate, a.result, a.err
	a.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return contract.Analysis{}, ctx.Err()
		}
	}
	return res, err
}

func (a *fakeAnalyzer) set(res contract.Analysis, err error, gate chan struct{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.result, a.err, a.gate = res, err, gate
}

// --- 테스트 환경 ---

type env struct {
	t        *testing.T
	srv      *httptest.Server
	mail     *fakeMailer
	analyzer *fakeAnalyzer
	app      *app.App
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := pgtest.New(t)
	e := &env{t: t, mail: &fakeMailer{code: map[string]string{}}, analyzer: &fakeAnalyzer{result: sampleAnalysis()}}
	cfg := config.Config{JWTSecret: "test-secret-at-least-16-chars"}
	e.app = app.Build(pool, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), app.Overrides{Analyzer: e.analyzer, Mailer: e.mail})
	e.srv = httptest.NewServer(e.app.Echo)
	t.Cleanup(func() {
		e.srv.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		e.app.Contracts.Shutdown(ctx)
	})
	return e
}

func sampleAnalysis() contract.Analysis {
	deposit := int64(300_000_000)
	return contract.Analysis{
		RiskScore: 72, Grade: contract.SeverityHigh, LossAmount: 50_000_000, LossMin: 20_000_000, LossMax: 90_000_000,
		Deposit: &deposit, ModelVersion: "gemma4:12b", PolicyVersion: "2026.10",
		Clauses: []contract.AnalysisClause{
			{Original: "첫 조항", ClauseType: "선순위근저당", Confidence: 0.9, Severity: contract.SeverityHigh, LMax: 100, ExpectedLoss: 40_000_000, Reason: "사유1", LawReference: []string{"민법 제1조", "민법 제2조"}, SafeText: "대체1"},
			{Original: "둘째 조항", ClauseType: "기타불이익", Confidence: 0.5, Severity: contract.SeverityLow, LMax: 5, ExpectedLoss: 10_000_000, Reason: "사유2", SafeText: "대체2"},
		},
	}
}

type resp struct {
	Status int
	Header http.Header
	Body   map[string]any
	Raw    []byte
}

func (e *env) do(method, path, token string, body io.Reader, contentType string, mods ...func(*http.Request)) resp {
	e.t.Helper()
	req, err := http.NewRequest(method, e.srv.URL+path, body)
	if err != nil {
		e.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for _, m := range mods {
		m(req)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := resp{Status: res.StatusCode, Header: res.Header, Raw: raw}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func (e *env) json(method, path, token string, payload any) resp {
	e.t.Helper()
	b, _ := json.Marshal(payload)
	return e.do(method, path, token, bytes.NewReader(b), "application/json")
}

func (e *env) get(path, token string) resp { return e.do("GET", path, token, nil, "") }

// login은 이메일로 가입·로그인하고 access 토큰을 돌려준다.
func (e *env) login(email string) string {
	e.t.Helper()
	if r := e.json("POST", "/api/v1/auth/email/code", "", map[string]any{"email": email}); r.Status != 200 {
		e.t.Fatalf("코드 요청: %d %s", r.Status, r.Raw)
	}
	r := e.json("POST", "/api/v1/auth/email/confirm", "", map[string]any{"email": email, "code": e.mail.last(email)})
	if r.Status != 200 {
		e.t.Fatalf("로그인: %d %s", r.Status, r.Raw)
	}
	return r.Body["accessToken"].(string)
}

func (e *env) upload(token, field, filename, mime string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	h.Set("Content-Type", mime)
	part, _ := w.CreatePart(h)
	part.Write(data)
	w.Close()
	return e.do("POST", "/api/v1/contracts", token, &buf, w.FormDataContentType())
}

func (e *env) uploadPDF(token string) string {
	e.t.Helper()
	r := e.upload(token, "file", "lease.pdf", "application/pdf", []byte("%PDF-1.4 test"))
	if r.Status != 202 {
		e.t.Fatalf("업로드: %d %s", r.Status, r.Raw)
	}
	return r.Body["id"].(string)
}

// waitDone은 분석이 pending을 벗어날 때까지 기다린다.
func (e *env) waitDone(token, id string) resp {
	e.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r := e.get("/api/v1/contracts/"+id, token)
		if r.Status == 200 && r.Body["status"] != "pending" {
			return r
		}
		time.Sleep(20 * time.Millisecond)
	}
	e.t.Fatal("분석이 끝나지 않았다")
	return resp{}
}

func wantErr(t *testing.T, r resp, status int, msg string) {
	t.Helper()
	if r.Status != status || r.Body["error"] != msg {
		t.Errorf("상태 %d, 메시지 %q를 기대했지만 %d %s", status, msg, r.Status, r.Raw)
	}
}

// --- 인증 ---

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)

	wantErr(t, e.json("POST", "/api/v1/auth/email/code", "", map[string]any{"email": "nope"}), 400, "이메일 형식이 올바르지 않아요")
	wantErr(t, e.do("POST", "/api/v1/auth/email/code", "", strings.NewReader("{bad"), "application/json"), 400, "요청 형식이 올바르지 않아요")

	r := e.json("POST", "/api/v1/auth/email/code", "", map[string]any{"email": "User@Example.com"})
	if r.Status != 200 || r.Body["retryAfterSeconds"] != float64(60) {
		t.Fatalf("코드 요청: %d %s", r.Status, r.Raw)
	}
	code := e.mail.last("user@example.com")
	if len(code) != 6 {
		t.Fatalf("6자리 코드를 보내야 한다: %q", code)
	}

	r = e.json("POST", "/api/v1/auth/email/code", "", map[string]any{"email": "user@example.com"})
	if r.Status != 429 || r.Header.Get("Retry-After") == "" {
		t.Errorf("쿨다운 안의 재요청은 429와 Retry-After: %d %v", r.Status, r.Header)
	}

	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	wantErr(t, e.json("POST", "/api/v1/auth/email/confirm", "", map[string]any{"email": "user@example.com", "code": wrong}), 400, "인증코드가 올바르지 않아요")

	r = e.json("POST", "/api/v1/auth/email/confirm", "", map[string]any{"email": "user@example.com", "code": code, "rememberDevice": true})
	if r.Status != 200 || r.Body["accessToken"] == "" || r.Body["refreshToken"] == "" {
		t.Fatalf("로그인: %d %s", r.Status, r.Raw)
	}
	cookies := r.Header.Values("Set-Cookie")
	if len(cookies) != 2 {
		t.Fatalf("쿠키 두 개(access, refresh)가 내려가야 한다: %v", cookies)
	}
	for _, c := range cookies {
		if !strings.Contains(c, "HttpOnly") || !strings.Contains(c, "SameSite=Lax") || !strings.Contains(c, "Path=/") {
			t.Errorf("쿠키 속성: %s", c)
		}
	}
	access, refresh := r.Body["accessToken"].(string), r.Body["refreshToken"].(string)

	// 코드는 한 번만 쓸 수 있다
	wantErr(t, e.json("POST", "/api/v1/auth/email/confirm", "", map[string]any{"email": "user@example.com", "code": code}), 400, "인증코드가 만료됐어요. 다시 받아 주세요")

	// Bearer와 쿠키 둘 다 인증된다. 이미 헤더가 있으면 쿠키로 넘어가지 않는다
	me := e.get("/api/v1/users/me", access)
	if me.Status != 200 || me.Body["email"] != "user@example.com" || me.Body["emailVerified"] != true || me.Body["name"] != "" || me.Body["id"] == "" {
		t.Errorf("내 정보: %d %s", me.Status, me.Raw)
	}
	viaCookie := e.do("GET", "/api/v1/users/me", "", nil, "", func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "cs_access", Value: access}) })
	if viaCookie.Status != 200 {
		t.Errorf("쿠키 인증: %d", viaCookie.Status)
	}
	badHeaderGoodCookie := e.do("GET", "/api/v1/users/me", "garbage", nil, "", func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "cs_access", Value: access}) })
	if badHeaderGoodCookie.Status != 401 {
		t.Errorf("틀린 Authorization 헤더가 있으면 쿠키로 넘어가지 않는다: %d", badHeaderGoodCookie.Status)
	}
	wantErr(t, e.get("/api/v1/users/me", ""), 401, "unauthorized")
	wantErr(t, e.get("/api/v1/users/me", refresh), 401, "unauthorized") // refresh 토큰으로는 API를 쓸 수 없다

	// refresh: 쿠키로도 본문으로도 갱신된다. refresh 토큰은 그대로다
	byCookie := e.do("POST", "/api/v1/auth/refresh", "", nil, "", func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "cs_refresh", Value: refresh}) })
	if byCookie.Status != 200 || byCookie.Body["refreshToken"] != refresh || byCookie.Body["accessToken"] == "" {
		t.Errorf("쿠키로 갱신: %d %s", byCookie.Status, byCookie.Raw)
	}
	byBody := e.json("POST", "/api/v1/auth/refresh", "", map[string]any{"refreshToken": refresh})
	if byBody.Status != 200 {
		t.Errorf("본문으로 갱신: %d %s", byBody.Status, byBody.Raw)
	}
	wantErr(t, e.json("POST", "/api/v1/auth/refresh", "", map[string]any{"refreshToken": access}), 401, "unauthorized")
	wantErr(t, e.do("POST", "/api/v1/auth/refresh", "", nil, ""), 401, "unauthorized")

	// 로그아웃하면 세션이 폐기돼 두 토큰 모두 쓸 수 없다
	out := e.do("POST", "/api/v1/auth/logout", access, nil, "")
	if out.Status != 204 {
		t.Fatalf("로그아웃: %d", out.Status)
	}
	if got := out.Header.Values("Set-Cookie"); len(got) != 2 || !strings.Contains(got[0], "Max-Age=0") {
		t.Errorf("쿠키를 지워야 한다: %v", got)
	}
	wantErr(t, e.get("/api/v1/users/me", access), 401, "unauthorized")
	wantErr(t, e.json("POST", "/api/v1/auth/refresh", "", map[string]any{"refreshToken": refresh}), 401, "unauthorized")
	if r := e.do("POST", "/api/v1/auth/logout", "", nil, ""); r.Status != 204 {
		t.Errorf("로그인하지 않아도 로그아웃은 204: %d", r.Status)
	}
}

func TestSameEmailKeepsSameUser(t *testing.T) {
	e := newEnv(t)
	first := e.get("/api/v1/users/me", e.login("same@x.com")).Body["id"]
	// 다시 로그인해도(대소문자가 달라도) 새로 가입되지 않고 같은 사용자다
	second := e.get("/api/v1/users/me", e.login("SAME@x.com")).Body["id"]
	if first == nil || first != second {
		t.Errorf("같은 이메일은 같은 사용자여야 한다: %v vs %v", first, second)
	}
}

// --- 사용자 · 동의 ---

func TestConsent(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")
	post := func(body any) resp { return e.json("POST", "/api/v1/users/me/consents", token, body) }

	wantErr(t, e.json("POST", "/api/v1/users/me/consents", "", map[string]any{"terms": true, "privacy": true}), 401, "unauthorized")
	wantErr(t, post(map[string]any{"terms": "yes", "privacy": true}), 400, "terms, privacy는 boolean이어야 합니다")
	wantErr(t, post(map[string]any{"privacy": true}), 400, "terms, privacy는 boolean이어야 합니다")
	wantErr(t, e.do("POST", "/api/v1/users/me/consents", token, strings.NewReader("not json"), "application/json"), 400, "terms, privacy는 boolean이어야 합니다")
	wantErr(t, post(map[string]any{"terms": true, "privacy": false}), 400, "이용약관과 개인정보처리방침 동의는 필수입니다")
	wantErr(t, post(map[string]any{"terms": false, "privacy": false}), 400, "이용약관과 개인정보처리방침 동의는 필수입니다")

	r := post(map[string]any{"terms": true, "privacy": true})
	if r.Status != 201 || r.Body["marketing"] != false || r.Body["terms"] != true || r.Body["privacy"] != true || r.Body["id"] == "" || r.Body["userId"] == "" || r.Body["createdAt"] == "" {
		t.Errorf("marketing 기본값 false: %d %s", r.Status, r.Raw)
	}
	r = post(map[string]any{"terms": true, "privacy": true, "marketing": true})
	if r.Status != 201 || r.Body["marketing"] != true {
		t.Errorf("marketing 동의: %d %s", r.Status, r.Raw)
	}
	r = post(map[string]any{"terms": true, "privacy": true, "marketing": "x"})
	if r.Status != 201 || r.Body["marketing"] != false {
		t.Errorf("marketing이 boolean이 아니면 false: %d %s", r.Status, r.Raw)
	}
}

// --- 계약서 ---

func TestUploadValidation(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")

	wantErr(t, e.upload("", "file", "a.pdf", "application/pdf", []byte("%PDF")), 401, "unauthorized")
	wantErr(t, e.upload(token, "other", "a.pdf", "application/pdf", []byte("%PDF")), 400, "file이 필요합니다")
	wantErr(t, e.do("POST", "/api/v1/contracts", token, strings.NewReader("x"), "application/json"), 400, "file이 필요합니다")
	wantErr(t, e.upload(token, "file", "a.txt", "text/plain", []byte("hello")), 415, "PDF 파일만 올릴 수 있어요")
	wantErr(t, e.upload(token, "file", "a.pdf", "application/pdf", nil), 413, "파일 크기는 10MB 이하여야 해요")
	wantErr(t, e.upload(token, "file", "big.pdf", "application/pdf", make([]byte, contract.MaxUploadBytes+1)), 413, "파일 크기는 10MB 이하여야 해요")
	wantErr(t, e.upload(token, "file", "huge.pdf", "application/pdf", make([]byte, contract.MaxUploadBytes+(2<<20))), 413, "파일 크기는 10MB 이하여야 해요")

	// 확장자나 MIME 중 하나만 PDF여도 받는다
	if r := e.upload(token, "file", "scan.PDF", "application/octet-stream", []byte("%PDF")); r.Status != 202 {
		t.Errorf("확장자로 판별: %d %s", r.Status, r.Raw)
	}
	if r := e.upload(token, "file", "noext", "application/pdf", []byte("%PDF")); r.Status != 202 {
		t.Errorf("MIME으로 판별: %d %s", r.Status, r.Raw)
	}
}

func TestContractCompletedShape(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")

	gate := make(chan struct{})
	e.analyzer.set(sampleAnalysis(), nil, gate)
	id := e.uploadPDF(token)

	// 분석 중: 200이고 progress만 값이 있다. 나머지 결과 필드는 null이지만 키는 있다
	pend := e.get("/api/v1/contracts/"+id, token)
	if pend.Status != 200 || pend.Body["status"] != "pending" || pend.Body["error"] != nil {
		t.Fatalf("pending 상세: %d %s", pend.Status, pend.Raw)
	}
	for _, k := range []string{"riskScore", "grade", "lossRange", "depositAmount", "flow"} {
		if v, ok := pend.Body[k]; !ok || v != nil {
			t.Errorf("pending의 %s는 null이어야 한다: %v (키 있음=%v)", k, v, ok)
		}
	}
	if p, _ := pend.Body["progress"].(map[string]any); p == nil || p["stage"] == "" {
		t.Errorf("pending이면 progress가 있다: %s", pend.Raw)
	}
	if w, ok := pend.Body["warnings"].([]any); !ok || len(w) != 0 {
		t.Errorf("warnings 기본값 []: %v", pend.Body["warnings"])
	}
	if items, ok := pend.Body["riskItems"].([]any); !ok || len(items) != 0 {
		t.Errorf("riskItems 기본값 []: %v", pend.Body["riskItems"])
	}
	if r := e.get("/api/v1/contracts/"+id+"/clauses/x", token); r.Status != 404 {
		t.Errorf("완료 전 조항 조회는 404: %d", r.Status)
	}

	close(gate)
	done := e.waitDone(token, id)
	b := done.Body
	if b["status"] != "completed" || b["progress"] != nil || b["error"] != nil {
		t.Fatalf("completed 상태: %s", done.Raw)
	}
	if b["riskScore"] != float64(72) || b["grade"] != "high" || b["lossAmount"] != float64(50_000_000) || b["depositAmount"] != float64(300_000_000) {
		t.Errorf("결과 값: %s", done.Raw)
	}
	if lr, _ := b["lossRange"].([]any); len(lr) != 2 || lr[0] != float64(20_000_000) || lr[1] != float64(90_000_000) {
		t.Errorf("lossRange: %v", b["lossRange"])
	}
	if b["flow"] != nil || b["fileName"] != "lease.pdf" {
		t.Errorf("flow/fileName: %v %v", b["flow"], b["fileName"])
	}

	items := b["riskItems"].([]any)
	if len(items) != 2 {
		t.Fatalf("조항 수: %d", len(items))
	}
	first, second := items[0].(map[string]any), items[1].(map[string]any)
	if first["original"] != "첫 조항" || second["original"] != "둘째 조항" {
		t.Errorf("orderIndex 순: %v / %v", first["original"], second["original"])
	}
	if first["lawReference"] != "민법 제1조, 민법 제2조" {
		t.Errorf("상세 응답의 lawReference는 쉼표로 이은 문자열이다: %#v", first["lawReference"])
	}
	if v, ok := second["lawReference"]; !ok || v != nil {
		t.Errorf("법령이 없으면 null: %#v", v)
	}
	if first["confidence"] != 0.9 || first["severity"] != "high" || first["clauseType"] != "선순위근저당" {
		t.Errorf("조항 값: %v", first)
	}

	// 조항 상세: lawReference는 배열이다
	clauseID := first["id"].(string)
	cl := e.get("/api/v1/contracts/"+id+"/clauses/"+clauseID, token)
	if cl.Status != 200 || cl.Body["contractId"] != id || cl.Body["safeText"] != "대체1" {
		t.Fatalf("조항 상세: %d %s", cl.Status, cl.Raw)
	}
	if law, _ := cl.Body["lawReference"].([]any); len(law) != 2 || law[1] != "민법 제2조" {
		t.Errorf("조항 상세의 lawReference는 배열: %v", cl.Body["lawReference"])
	}
	cl2 := e.get("/api/v1/contracts/"+id+"/clauses/"+second["id"].(string), token)
	if law, ok := cl2.Body["lawReference"].([]any); !ok || len(law) != 0 {
		t.Errorf("법령이 없으면 []: %#v", cl2.Body["lawReference"])
	}
	wantErr(t, e.get("/api/v1/contracts/"+id+"/clauses/nope", token), 404, "not found")
}

func TestContractFailedShape(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")
	e.analyzer.set(contract.Analysis{}, &contractsvc.AnalysisError{Message: "글자를 읽지 못했어요"}, nil)
	id := e.uploadPDF(token)

	done := e.waitDone(token, id)
	if done.Body["status"] != "failed" || done.Body["error"] != "글자를 읽지 못했어요" || done.Body["progress"] != nil {
		t.Errorf("failed 상태: %s", done.Raw)
	}
	wantErr(t, e.get("/api/v1/contracts/"+id+"/clauses/x", token), 404, "not found")
}

func TestOwnershipIsolation(t *testing.T) {
	e := newEnv(t)
	owner, other := e.login("owner@x.com"), e.login("other@x.com")
	id := e.uploadPDF(owner)
	done := e.waitDone(owner, id)
	clauseID := done.Body["riskItems"].([]any)[0].(map[string]any)["id"].(string)

	wantErr(t, e.get("/api/v1/contracts/"+id, other), 404, "not found")
	wantErr(t, e.get("/api/v1/contracts/"+id+"/clauses/"+clauseID, other), 404, "not found")
	wantErr(t, e.do("DELETE", "/api/v1/contracts/"+id, other, nil, ""), 404, "not found")
	if r := e.get("/api/v1/contracts", other); r.Body["total"] != float64(0) {
		t.Errorf("남의 계약서는 목록에도 없다: %s", r.Raw)
	}
	if r := e.get("/api/v1/contracts/"+id, owner); r.Status != 200 {
		t.Errorf("소유자는 그대로 볼 수 있다: %d", r.Status)
	}
}

func TestUnauthenticatedContractEndpoints(t *testing.T) {
	e := newEnv(t)
	for _, c := range [][2]string{
		{"GET", "/api/v1/contracts"}, {"GET", "/api/v1/contracts/x"}, {"DELETE", "/api/v1/contracts/x"}, {"GET", "/api/v1/contracts/x/clauses/y"},
	} {
		wantErr(t, e.do(c[0], c[1], "", nil, ""), 401, "unauthorized")
	}
}

func TestDelete(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")
	id := e.uploadPDF(token)
	e.waitDone(token, id)

	if r := e.do("DELETE", "/api/v1/contracts/"+id, token, nil, ""); r.Status != 204 || len(r.Raw) != 0 {
		t.Errorf("삭제: %d %q", r.Status, r.Raw)
	}
	wantErr(t, e.get("/api/v1/contracts/"+id, token), 404, "not found")
	wantErr(t, e.do("DELETE", "/api/v1/contracts/"+id, token, nil, ""), 404, "not found")
}

func TestDeleteDuringAnalysisDiscardsResult(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")
	gate := make(chan struct{})
	e.analyzer.set(sampleAnalysis(), nil, gate)
	id := e.uploadPDF(token)

	if r := e.do("DELETE", "/api/v1/contracts/"+id, token, nil, ""); r.Status != 204 {
		t.Fatalf("분석 중 삭제: %d", r.Status)
	}
	close(gate)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	e.app.Contracts.Shutdown(ctx) // 분석 작업이 끝날 때까지 기다린다
	wantErr(t, e.get("/api/v1/contracts/"+id, token), 404, "not found")
}

func TestListContracts(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")

	// 완료(조항 2개) → 실패 → 진행 중 순서로 만든다
	done := e.uploadPDF(token)
	e.waitDone(token, done)
	e.analyzer.set(contract.Analysis{}, &contractsvc.AnalysisError{Message: "실패 사유"}, nil)
	failed := e.uploadPDF(token)
	e.waitDone(token, failed)
	gate := make(chan struct{})
	e.analyzer.set(sampleAnalysis(), nil, gate)
	pending := e.uploadPDF(token)
	t.Cleanup(func() { close(gate) })

	r := e.get("/api/v1/contracts", token)
	items := r.Body["items"].([]any)
	if r.Status != 200 || len(items) != 3 || r.Body["page"] != float64(1) || r.Body["total"] != float64(3) || r.Body["hasMore"] != false {
		t.Fatalf("목록: %d %s", r.Status, r.Raw)
	}
	byID := map[string]map[string]any{}
	for _, it := range items {
		m := it.(map[string]any)
		byID[m["id"].(string)] = m
	}
	if items[0].(map[string]any)["id"] != pending || items[2].(map[string]any)["id"] != done {
		t.Errorf("최신순이어야 한다: %v", items)
	}
	if byID[done]["riskCount"] != float64(2) || byID[done]["error"] != nil || byID[done]["lossAmount"] != float64(50_000_000) {
		t.Errorf("완료 항목: %v", byID[done])
	}
	if v, ok := byID[failed]["riskCount"]; !ok || v != nil || byID[failed]["error"] != "실패 사유" {
		t.Errorf("실패 항목은 riskCount가 null이고 error가 있다: %v", byID[failed])
	}
	if v, ok := byID[pending]["riskCount"]; !ok || v != nil || byID[pending]["error"] != nil {
		t.Errorf("진행 중 항목: %v", byID[pending])
	}

	// 페이지네이션과 보정
	p1 := e.get("/api/v1/contracts?page=1&limit=2", token)
	if len(p1.Body["items"].([]any)) != 2 || p1.Body["hasMore"] != true {
		t.Errorf("1쪽: %s", p1.Raw)
	}
	p2 := e.get("/api/v1/contracts?page=2&limit=2", token)
	if len(p2.Body["items"].([]any)) != 1 || p2.Body["hasMore"] != false || p2.Body["page"] != float64(2) {
		t.Errorf("2쪽: %s", p2.Raw)
	}
	if r := e.get("/api/v1/contracts?page=abc&limit=0", token); r.Body["page"] != float64(1) || len(r.Body["items"].([]any)) != 3 {
		t.Errorf("잘못된 값은 기본값: %s", r.Raw)
	}
	if r := e.get("/api/v1/contracts?page=-3&limit=-5", token); r.Body["page"] != float64(1) || len(r.Body["items"].([]any)) != 1 {
		t.Errorf("음수는 1로 올린다(limit=1): %s", r.Raw)
	}
	if r := e.get("/api/v1/contracts?limit=1000", token); len(r.Body["items"].([]any)) != 3 {
		t.Errorf("limit 상한 50: %s", r.Raw)
	}

	// 홈 화면용: 실패를 제외한다
	active := e.get("/api/v1/contracts?status=active&limit=3", token)
	ai := active.Body["items"].([]any)
	if len(ai) != 2 || ai[0].(map[string]any)["id"] != pending || ai[1].(map[string]any)["id"] != done {
		t.Errorf("status=active는 완료·진행 중만: %s", active.Raw)
	}
	if one := e.get("/api/v1/contracts?status=active&limit=1", token); len(one.Body["items"].([]any)) != 1 {
		t.Errorf("limit 적용: %s", one.Raw)
	}
}

func TestStalePendingIsMarkedFailedOnRead(t *testing.T) {
	e := newEnv(t)
	token := e.login("a@b.com")
	gate := make(chan struct{})
	e.analyzer.set(sampleAnalysis(), nil, gate)
	t.Cleanup(func() { close(gate) })
	id := e.uploadPDF(token)

	// 16분 전에 만든 것처럼 만든다
	ctx := context.Background()
	if _, err := e.app.PoolForTest().Exec(ctx, `UPDATE contract SET created_at = now() - interval '16 minutes' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	r := e.get("/api/v1/contracts/"+id, token)
	if r.Body["status"] != "failed" || r.Body["error"] != "분석이 중단됐어요. 다시 올려 주세요" {
		t.Errorf("오래된 pending은 failed로 바뀐다: %s", r.Raw)
	}
}

func TestHealthAndUnknownRoute(t *testing.T) {
	e := newEnv(t)
	if r := e.get("/healthz", ""); r.Status != 200 || r.Body["status"] != "ok" {
		t.Errorf("healthz: %d %s", r.Status, r.Raw)
	}
	r := e.get("/api/v1/nope", "")
	if r.Status != 404 || r.Body["error"] == nil {
		t.Errorf("없는 경로도 {error} 모양: %d %s", r.Status, r.Raw)
	}
}
