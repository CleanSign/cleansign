// Package mail 은 로그인 OTP 메일을 SMTP 로 보내는 outbound 어댑터다.
// service/auth.Mailer 포트를 구현한다 (import 없이 구조적으로 만족).
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
	// 컨텍스트에 deadline 이 없을 때 쓰는 클라이언트 측 타임아웃.
	defaultTimeout = 15 * time.Second
	// 웹(TS) 코드와 동일하게 예시 호스트는 "미설정" 으로 취급한다.
	placeholderHost = "smtp.example.com"
)

// Config 는 SMTP 접속 정보다. Gmail 은 앱 비밀번호를 Password 에 넣는다.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// Sender 는 SMTP 로 OTP 메일을 보낸다.
type Sender struct {
	cfg  Config
	log  *slog.Logger
	send func(ctx context.Context, msg *gomail.Msg) error
}

// New 는 Sender 를 만든다. log 가 nil 이면 slog.Default 를 쓴다.
func New(cfg Config, log *slog.Logger) *Sender {
	if log == nil {
		log = slog.Default()
	}
	s := &Sender{cfg: cfg, log: log}
	s.send = s.dialAndSend
	return s
}

// Configured 는 실제 SMTP 로 보낼 수 있는지 여부다.
func (s *Sender) Configured() bool {
	return s.cfg.Host != "" && s.cfg.Host != placeholderHost
}

// SendCode 는 로그인 코드를 to 로 보낸다.
// SMTP 미설정(개발 환경)이면 보내지 않고 코드를 로그로만 남긴다.
func (s *Sender) SendCode(ctx context.Context, to, code string, ttl time.Duration) error {
	if !s.Configured() {
		// 개발자가 서버 로그에서 코드를 읽을 수 있게 한다. 운영(설정됨)에선 코드를 로그에 남기지 않는다.
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

// dialAndSend 는 실제 SMTP 전송이다. 에러에 비밀번호가 섞이지 않도록 그대로 감싸 반환만 한다.
func (s *Sender) dialAndSend(ctx context.Context, msg *gomail.Msg) error {
	opts := []gomail.Option{
		gomail.WithPort(s.cfg.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
		gomail.WithUsername(s.cfg.User),
		gomail.WithPassword(s.cfg.Password),
		gomail.WithTimeout(defaultTimeout),
	}
	if s.cfg.Port == 465 {
		// 465 는 암묵적 TLS (TS 의 secure: port === 465).
		opts = append(opts, gomail.WithSSL())
	} else {
		// 그 외는 STARTTLS 필수: 평문으로 자격증명이 나가는 것을 막는다.
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

// html/template 이라 code 는 자동 이스케이프된다.
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

// ttlMinutes 는 TTL 을 안내용 분 단위로 바꾼다. 1분 미만이어도 "0분" 이 되지 않게 최소 1.
func ttlMinutes(ttl time.Duration) int {
	if m := int(ttl / time.Minute); m > 0 {
		return m
	}
	return 1
}

// buildMessage 는 네트워크 없이 메시지만 만드는 순수 함수다 (테스트 용이).
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
