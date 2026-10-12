package mail

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"mime"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	gomail "github.com/wneessen/go-mail"
)

// A local interface identical to service/auth.Mailer (not imported because of the architecture linter).
type mailer interface {
	SendCode(ctx context.Context, to, code string, ttl time.Duration) error
}

var _ mailer = (*Sender)(nil)

func partsOf(t *testing.T, m *gomail.Msg) (text, html string) {
	t.Helper()
	for _, p := range m.GetParts() {
		b, err := p.GetContent()
		if err != nil {
			t.Fatal(err)
		}
		switch p.GetContentType() {
		case gomail.TypeTextPlain:
			text = string(b)
		case gomail.TypeTextHTML:
			html = string(b)
		}
	}
	return
}

func TestBuildMessage(t *testing.T) {
	cfg := Config{User: "user@example.org", From: "Clean Sign <noreply@example.org>"}
	m, err := buildMessage(cfg, "a@b.com", "123456", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// The subject is stored encoded as RFC 2047, so decode it before comparing.
	raw := m.GetGenHeader(gomail.HeaderSubject)
	if len(raw) != 1 {
		t.Fatalf("subject = %v", raw)
	}
	if subj, err := new(mime.WordDecoder).DecodeHeader(raw[0]); err != nil || subj != "[Clean Sign] 인증코드 123456" {
		t.Errorf("subject = %q (%v)", raw[0], err)
	}
	if got := m.GetFromString(); len(got) != 1 || !strings.Contains(got[0], "noreply@example.org") {
		t.Errorf("from = %v", got)
	}
	if got := m.GetToString(); len(got) != 1 || !strings.Contains(got[0], "a@b.com") {
		t.Errorf("to = %v", got)
	}
	text, html := partsOf(t, m)
	if text == "" || html == "" {
		t.Fatalf("both parts required: text=%q html=%q", text, html)
	}
	for _, s := range []string{"123456", "10분 후 만료됩니다", "아래 인증코드를 입력해 로그인을 완료하세요."} {
		if !strings.Contains(text, s) || !strings.Contains(html, s) {
			t.Errorf("missing %q", s)
		}
	}
	for _, s := range []string{"font-size: 28px", "font-weight: 700", "letter-spacing: 4px", "<h2>Clean Sign 인증코드</h2>"} {
		if !strings.Contains(html, s) {
			t.Errorf("html missing %q", s)
		}
	}
}

func TestBuildMessageFromFallback(t *testing.T) {
	m, err := buildMessage(Config{User: "user@example.org"}, "a@b.com", "1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.GetFromString(); len(got) != 1 || !strings.Contains(got[0], "user@example.org") {
		t.Errorf("from = %v", got)
	}
}

func TestBuildMessageEscapesHTML(t *testing.T) {
	m, err := buildMessage(Config{User: "u@example.org"}, "a@b.com", "<b>x</b>", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	_, html := partsOf(t, m)
	if strings.Contains(html, "<b>x</b>") || !strings.Contains(html, "&lt;b&gt;") {
		t.Errorf("code not escaped: %s", html)
	}
	// An invalid recipient address must be an error (prevents header injection).
	if _, err := buildMessage(Config{User: "u@example.org"}, "<b>@x\r\nBcc: e@e.com", "1", time.Minute); err == nil {
		t.Error("adversarial to should fail")
	}
}

func TestSendCodeConfigured(t *testing.T) {
	s := New(Config{Host: "smtp.gmail.com", Port: 587, User: "u@example.org", Password: "secret"}, slog.New(slog.DiscardHandler))
	var calls int
	var got *gomail.Msg
	s.send = func(_ context.Context, m *gomail.Msg) error { calls++; got = m; return nil }
	if err := s.SendCode(context.Background(), "a@b.com", "654321", 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
	text, _ := partsOf(t, got)
	if !strings.Contains(text, "654321") || !strings.Contains(text, "5분") {
		t.Errorf("text = %q", text)
	}
}

func TestSendCodeError(t *testing.T) {
	s := New(Config{Host: "smtp.gmail.com", Port: 587, User: "u@example.org"}, slog.New(slog.DiscardHandler))
	boom := errors.New("boom")
	s.send = func(context.Context, *gomail.Msg) error { return boom }
	err := s.SendCode(context.Background(), "a@b.com", "1", time.Minute)
	if !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "send mail: ") {
		t.Errorf("err = %v", err)
	}
}

func TestSendCodeDevMode(t *testing.T) {
	for _, host := range []string{"", "smtp.example.com"} {
		var buf bytes.Buffer
		s := New(Config{Host: host}, slog.New(slog.NewTextHandler(&buf, nil)))
		s.send = func(context.Context, *gomail.Msg) error { t.Error("send must not be called"); return nil }
		if s.Configured() {
			t.Errorf("host %q should be unconfigured", host)
		}
		if err := s.SendCode(context.Background(), "a@b.com", "246810", 10*time.Minute); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		for _, want := range []string{"[dev] login code", "246810", "a@b.com", "level=INFO"} {
			if !strings.Contains(out, want) {
				t.Errorf("host %q: log missing %q: %s", host, want, out)
			}
		}
	}
}

// TestSendRealMail sends one real mail. Skipped unless MAIL_TEST_TO is set, e.g.
//
//	set -a; . ../../../.env; set +a; MAIL_TEST_TO=me@example.com go test -run TestSendRealMail -v .
//
// Then check the inbox *and* the spam folder.
func TestSendRealMail(t *testing.T) {
	to := os.Getenv("MAIL_TEST_TO")
	if to == "" {
		t.Skip("MAIL_TEST_TO not set")
	}
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	s := New(Config{Host: os.Getenv("SMTP_HOST"), Port: port, User: os.Getenv("SMTP_USER"), Password: os.Getenv("SMTP_PASSWORD"), From: os.Getenv("SMTP_FROM")}, nil)
	if !s.Configured() {
		t.Fatal("SMTP_HOST not set")
	}
	if err := s.SendCode(context.Background(), to, "123456", 5*time.Minute); err != nil {
		t.Fatal(err)
	}
}
