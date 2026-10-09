// Package ai는 Python ai-service를 호출하는 나가는 어댑터다. Analyzer 포트를 구현한다.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	domain "github.com/CleanSign/cleansign/server/internal/domain/contract"
	svc "github.com/CleanSign/cleansign/server/internal/service/contract"
)

// 사용자에게 그대로 보여 주는 메시지들.
const (
	msgConnect   = "분석 서버에 연결하지 못했어요"
	msgTimeout   = "분석이 너무 오래 걸려 중단했어요. 잠시 후 다시 시도해 주세요"
	msgCanceled  = "분석이 중단됐어요. 다시 올려 주세요"
	msgUnreadble = "계약서를 읽을 수 없어요"
	msgGeneric   = "분석 중 문제가 생겼어요"
	msgNoResult  = "분석 결과를 받지 못했어요"
)

var _ svc.Analyzer = (*Client)(nil)

// Client는 ai-service HTTP 클라이언트다.
type Client struct {
	baseURL string
	token   string
	timeout time.Duration
	http    *http.Client
}

// Option은 Client 설정을 바꾼다.
type Option func(*Client)

// WithHTTPClient는 테스트 등에서 http.Client를 갈아 끼운다.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// New는 클라이언트를 만든다. timeout은 요청 전체(스트림 읽기 포함)의 제한 시간이다.
func New(baseURL, internalToken string, timeout time.Duration, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   internalToken,
		timeout: timeout,
		// 제한 시간은 context로 걸기 때문에 http.Client.Timeout은 쓰지 않는다.
		http: &http.Client{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Analyze는 PDF를 보내고 NDJSON 스트림을 읽으며 진행 상황을 알린 뒤 결과를 돌려준다.
// 실패는 모두 *svc.AnalysisError다. 단, onProgress가 돌려준 오류는 그대로 돌려준다.
func (c *Client) Analyze(ctx context.Context, fileName string, pdf []byte, onProgress svc.ProgressFunc) (domain.Analysis, error) {
	var zero domain.Analysis

	reqCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	body, contentType, err := buildMultipart(fileName, pdf)
	if err != nil {
		return zero, &svc.AnalysisError{Message: msgGeneric, Err: err}
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.baseURL+"/v1/analyze/stream", body)
	if err != nil {
		return zero, &svc.AnalysisError{Message: msgGeneric, Err: err}
	}
	req.Header.Set("Content-Type", contentType)
	if c.token != "" {
		req.Header.Set("X-Internal-Token", c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return zero, transportError(ctx, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode == http.StatusUnprocessableEntity {
		// 스캔본 등 읽지 못한 경우. detail은 사용자에게 그대로 보여 줘도 된다.
		var b struct {
			Detail any `json:"detail"`
		}
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		_ = json.Unmarshal(raw, &b)
		msg := msgUnreadble
		if s, ok := b.Detail.(string); ok && s != "" {
			msg = s
		}
		return zero, &svc.AnalysisError{Message: msg, Err: fmt.Errorf("ai-service 422: %s", snippet(raw))}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<10))
		return zero, &svc.AnalysisError{Message: msgGeneric, Err: fmt.Errorf("ai-service %d: %s", res.StatusCode, snippet(raw))}
	}

	// 결과 이벤트에 분석 전체가 실려 한 줄이 수백 KB가 될 수 있어 Scanner 대신 Reader를 쓴다.
	r := bufio.NewReaderSize(res.Body, 64<<10)
	for {
		line, rerr := r.ReadBytes('\n')
		if line = bytes.TrimSpace(line); len(line) > 0 {
			var ev event
			if err := json.Unmarshal(line, &ev); err != nil {
				return zero, transportError(ctx, fmt.Errorf("ndjson 파싱 실패: %w", err))
			}
			switch ev.Type {
			case "progress":
				if onProgress != nil {
					if err := onProgress(domain.Progress{Stage: ev.Stage, Done: ev.Done, Total: ev.Total}); err != nil {
						return zero, err
					}
				}
			case "result":
				a, err := decodeAnalysis(ev.Data)
				if err != nil {
					return zero, transportError(ctx, err)
				}
				return a, nil
			case "error":
				msg := msgGeneric
				if ev.Readable && ev.Detail != "" {
					msg = ev.Detail
				}
				return zero, &svc.AnalysisError{Message: msg, Err: fmt.Errorf("ai-service error 이벤트: %s", ev.Detail)}
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			return zero, transportError(ctx, rerr)
		}
	}
	// 정상 종료로 보여도 제한 시간이 지났다면 시간 초과가 원인이다.
	if reqCtx.Err() != nil {
		return zero, transportError(ctx, reqCtx.Err())
	}
	return zero, &svc.AnalysisError{Message: msgNoResult}
}

// buildMultipart는 file 파트 하나짜리 multipart 본문을 메모리에 만든다(PDF는 10MB 이하).
func buildMultipart(fileName string, pdf []byte) (*bytes.Reader, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeQuotes(fileName)))
	h.Set("Content-Type", "application/pdf")
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(pdf); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return bytes.NewReader(buf.Bytes()), w.FormDataContentType(), nil
}

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\r", "", "\n", "")

func escapeQuotes(s string) string { return quoteEscaper.Replace(s) }

// transportError는 전송·읽기 오류를 사용자 메시지로 분류한다.
// 호출한 쪽이 취소했다면 중단, 제한 시간이면 시간 초과, 나머지는 연결 실패다.
func transportError(parent context.Context, err error) error {
	if errors.Is(parent.Err(), context.Canceled) {
		return &svc.AnalysisError{Message: msgCanceled, Err: parent.Err()}
	}
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		return &svc.AnalysisError{Message: msgTimeout, Err: err}
	}
	return &svc.AnalysisError{Message: msgConnect, Err: err}
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 500 {
		s = s[:500] + "..."
	}
	return s
}

// ---- 와이어 형식 ----

type event struct {
	Type     string          `json:"type"`
	Stage    string          `json:"stage"`
	Done     int             `json:"done"`
	Total    int             `json:"total"`
	Data     json.RawMessage `json:"data"`
	Detail   string          `json:"detail"`
	Readable bool            `json:"readable"`
}

type wireAnalysis struct {
	RiskScore  float64   `json:"riskScore"`
	Grade      string    `json:"grade"`
	LossAmount float64   `json:"lossAmount"`
	LossRange  []float64 `json:"lossRange"`
	Facts      struct {
		Deposit *float64 `json:"deposit"`
	} `json:"facts"`
	Clauses       []wireClause    `json:"clauses"`
	Flow          json.RawMessage `json:"flow"`
	Warnings      []string        `json:"warnings"`
	ModelVersion  string          `json:"modelVersion"`
	PolicyVersion string          `json:"policyVersion"`
}

type wireClause struct {
	Original     string   `json:"original"`
	ClauseType   string   `json:"clauseType"`
	Confidence   float64  `json:"confidence"`
	Severity     string   `json:"severity"`
	LMax         float64  `json:"lMax"`
	ExpectedLoss float64  `json:"expectedLoss"`
	Reason       string   `json:"reason"`
	LawReference []string `json:"lawReference"`
	SafeText     string   `json:"safeText"`
}

// round는 TS의 Math.round처럼 금액을 원 단위 정수로 바꾼다.
func round(v float64) int64 { return int64(math.Round(v)) }

func decodeAnalysis(raw json.RawMessage) (domain.Analysis, error) {
	var w wireAnalysis
	if len(raw) == 0 {
		return domain.Analysis{}, errors.New("result 이벤트에 data가 없음")
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		return domain.Analysis{}, fmt.Errorf("분석 결과 해석 실패: %w", err)
	}
	a := domain.Analysis{
		RiskScore:     int32(math.Round(w.RiskScore)),
		Grade:         domain.Severity(w.Grade),
		LossAmount:    round(w.LossAmount),
		Warnings:      w.Warnings,
		ModelVersion:  w.ModelVersion,
		PolicyVersion: w.PolicyVersion,
	}
	if len(w.LossRange) >= 2 {
		a.LossMin, a.LossMax = round(w.LossRange[0]), round(w.LossRange[1])
	}
	if w.Facts.Deposit != nil {
		d := round(*w.Facts.Deposit)
		a.Deposit = &d
	}
	// JSON null은 RawMessage에 "null"로 남으므로 nil로 되돌린다.
	if f := bytes.TrimSpace(w.Flow); len(f) > 0 && !bytes.Equal(f, []byte("null")) {
		a.Flow = append(json.RawMessage(nil), f...)
	}
	a.Clauses = make([]domain.AnalysisClause, len(w.Clauses))
	for i, c := range w.Clauses {
		a.Clauses[i] = domain.AnalysisClause{
			Original:     c.Original,
			ClauseType:   c.ClauseType,
			Confidence:   c.Confidence,
			Severity:     domain.Severity(c.Severity),
			LMax:         round(c.LMax),
			ExpectedLoss: round(c.ExpectedLoss),
			Reason:       c.Reason,
			LawReference: c.LawReference,
			SafeText:     c.SafeText,
		}
	}
	return a, nil
}
