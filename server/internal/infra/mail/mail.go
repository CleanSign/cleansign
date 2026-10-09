// Package mail is the outbound adapter that sends login OTP emails over SMTP.
// It implements the service/auth.Mailer port (satisfied structurally, without importing it).
package mail

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	gomail "github.com/wneessen/go-mail"
)

const (
	subject = "Clean Sign 인증코드"
	// Client-side timeout used when the context has no deadline.
	defaultTimeout = 15 * time.Second
	// As in the web (TS) code, the example host is treated as "not configured".
	placeholderHost = "smtp.example.com"
)

// Config is the SMTP connection info. For Gmail, put the app password in Password.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// Sender sends OTP emails over SMTP.
type Sender struct {
	cfg  Config
	log  *slog.Logger
	send func(ctx context.Context, msg *gomail.Msg) error
}

// New creates a Sender. If log is nil, slog.Default is used.
func New(cfg Config, log *slog.Logger) *Sender {
	if log == nil {
		log = slog.Default()
	}
	s := &Sender{cfg: cfg, log: log}
	s.send = s.dialAndSend
	return s
}

// Configured reports whether it can send through a real SMTP server.
func (s *Sender) Configured() bool {
	return s.cfg.Host != "" && s.cfg.Host != placeholderHost
}

// SendCode sends the login code to to.
// If SMTP is not configured (development), it does not send and only logs the code.
func (s *Sender) SendCode(ctx context.Context, to, code string, ttl time.Duration) error {
	if !s.Configured() {
		// Lets developers read the code from the server log. In production (configured), the code is not logged.
		s.log.Info("[dev] login code", "to", to, "code", code, "ttl", ttl)
		return nil
	}
	msg, err := buildMessage(s.cfg, to, code, ttl)
	if err != nil {
		return fmt.Errorf("build mail: %w", err)
	}
	if err := s.send(ctx, msg); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

// dialAndSend is the actual SMTP delivery. It only wraps and returns errors as-is so the password does not leak into them.
func (s *Sender) dialAndSend(ctx context.Context, msg *gomail.Msg) error {
	opts := []gomail.Option{
		gomail.WithPort(s.cfg.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
		gomail.WithUsername(s.cfg.User),
		gomail.WithPassword(s.cfg.Password),
		gomail.WithTimeout(defaultTimeout),
	}
	if s.cfg.Port == 465 {
		// 465 is implicit TLS (TS's secure: port === 465).
		opts = append(opts, gomail.WithSSL())
	} else {
		// Otherwise STARTTLS is required: this prevents credentials from going out in plaintext.
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}
	c, err := gomail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}
	return c.DialAndSendWithContext(ctx, msg)
}

type bodyData struct {
	Code    string
	Minutes int
}

// Since this is html/template, code is auto-escaped.
var htmlTmpl = template.Must(template.New("otp").Parse(`<!DOCTYPE html>
<html>
<body>
<h2>Clean Sign 인증코드</h2>
<p>아래 인증코드를 입력해 로그인을 완료하세요.</p>
<p style="font-size: 28px; font-weight: 700; letter-spacing: 4px;">{{.Code}}</p>
<p>이 코드는 {{.Minutes}}분 후 만료됩니다. 본인이 요청하지 않았다면 이 메일을 무시하세요.</p>
</body>
</html>
`))

const textFmt = `Clean Sign 인증코드

아래 인증코드를 입력해 로그인을 완료하세요.

%s

이 코드는 %d분 후 만료됩니다. 본인이 요청하지 않았다면 이 메일을 무시하세요.
`

// ttlMinutes converts the TTL to minutes for display. At least 1, so it never becomes "0 minutes" even if under a minute.
func ttlMinutes(ttl time.Duration) int {
	if m := int(ttl / time.Minute); m > 0 {
		return m
	}
	return 1
}

// buildMessage is a pure function that builds the message without the network (easy to test).
func buildMessage(cfg Config, to, code string, ttl time.Duration) (*gomail.Msg, error) {
	from := cfg.From
	if from == "" {
		from = cfg.User
	}
	d := bodyData{Code: code, Minutes: ttlMinutes(ttl)}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, d); err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}

	msg := gomail.NewMsg()
	if err := msg.From(from); err != nil {
		return nil, fmt.Errorf("set from: %w", err)
	}
	if err := msg.To(to); err != nil {
		return nil, fmt.Errorf("set to: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(gomail.TypeTextPlain, fmt.Sprintf(textFmt, code, d.Minutes))
	msg.AddAlternativeString(gomail.TypeTextHTML, htmlBuf.String())
	return msg, nil
}
