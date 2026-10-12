package auth

import (
	"errors"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func TestNewCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		c, err := NewCode()
		if err != nil || len(c) != CodeLength {
			t.Fatalf("NewCode() = %q, %v", c, err)
		}
		for _, r := range c {
			if r < '0' || r > '9' {
				t.Fatalf("숫자가 아닌 문자: %q", c)
			}
		}
		seen[c] = true
	}
	if len(seen) < 10 {
		t.Errorf("무작위여야 한다 (서로 다른 코드 %d개)", len(seen))
	}
}

func TestHashCodeBindsEmail(t *testing.T) {
	a := HashCode("s", "a@x.com", "123456")
	if a != HashCode("s", "a@x.com", "123456") {
		t.Error("같은 입력은 같은 해시여야 한다")
	}
	if a == HashCode("s", "b@x.com", "123456") {
		t.Error("이메일이 다르면 해시가 달라야 한다")
	}
	if a == HashCode("other", "a@x.com", "123456") {
		t.Error("비밀키가 다르면 해시가 달라야 한다")
	}
}

func TestCanResend(t *testing.T) {
	c := EmailCode{SentAt: t0}
	if wait, ok := c.CanResend(t0.Add(30 * time.Second)); ok || wait != 30*time.Second {
		t.Errorf("30초 뒤에는 아직 안 된다: %v %v", wait, ok)
	}
	if _, ok := c.CanResend(t0.Add(ResendCooldown)); !ok {
		t.Error("쿨다운이 지나면 된다")
	}
}

func TestCheck(t *testing.T) {
	good := HashCode("s", "a@x.com", "123456")
	c := EmailCode{Email: "a@x.com", CodeHash: good, SentAt: t0, ExpiresAt: t0.Add(CodeTTL)}

	if err := c.Check(t0.Add(time.Minute), good); err != nil {
		t.Errorf("맞는 코드: %v", err)
	}
	if err := c.Check(t0.Add(time.Minute), HashCode("s", "a@x.com", "000000")); !errors.Is(err, ErrCodeInvalid) {
		t.Errorf("틀린 코드: %v", err)
	}
	if err := c.Check(t0.Add(CodeTTL), good); !errors.Is(err, ErrCodeExpired) {
		t.Errorf("만료 시각 이후: %v", err)
	}
	c.Attempts = MaxAttempts
	if err := c.Check(t0.Add(time.Minute), good); !errors.Is(err, ErrTooManyAttempts) {
		t.Errorf("시도 초과는 맞는 코드여도 거절한다: %v", err)
	}
}

func TestRefreshTTLFor(t *testing.T) {
	if RefreshTTLFor(false) != 72*time.Hour || RefreshTTLFor(true) != 30*24*time.Hour {
		t.Error("refresh 수명")
	}
}

func TestSessionIsActive(t *testing.T) {
	s := Session{ExpiresAt: t0}
	if !s.IsActive(t0.Add(-time.Second)) || s.IsActive(t0) {
		t.Error("만료 시각 전까지만 유효하다")
	}
}

func TestCanResendWindowLimit(t *testing.T) {
	full := EmailCode{SentAt: t0, SendCount: MaxSendsPerWindow, WindowStartedAt: t0.Add(-30 * time.Minute)}
	// Cooldown passed, but the window is full: wait until the window ends.
	if wait, ok := full.CanResend(t0.Add(ResendCooldown)); ok || wait != 30*time.Minute-ResendCooldown {
		t.Errorf("full window must block until it ends: %v %v", wait, ok)
	}
	if _, ok := full.CanResend(t0.Add(30 * time.Minute)); !ok {
		t.Error("a new window starts once the old one ended")
	}
	almost := EmailCode{SentAt: t0, SendCount: MaxSendsPerWindow - 1, WindowStartedAt: t0}
	if _, ok := almost.CanResend(t0.Add(ResendCooldown)); !ok {
		t.Error("below the limit it is allowed")
	}
}

func TestNextSend(t *testing.T) {
	if n, w := (EmailCode{}).NextSend(t0); n != 1 || !w.Equal(t0) {
		t.Errorf("first send starts a window: %d %v", n, w)
	}
	c := EmailCode{SendCount: 2, WindowStartedAt: t0}
	if n, w := c.NextSend(t0.Add(10 * time.Minute)); n != 3 || !w.Equal(t0) {
		t.Errorf("inside the window it counts up: %d %v", n, w)
	}
	if n, w := c.NextSend(t0.Add(SendWindow)); n != 1 || !w.Equal(t0.Add(SendWindow)) {
		t.Errorf("after the window it restarts: %d %v", n, w)
	}
}
