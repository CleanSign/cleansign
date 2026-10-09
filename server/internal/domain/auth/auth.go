// Package auth는 이메일 인증코드와 세션의 순수한 규칙을 담는다.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math/big"
	"time"
)

const (
	CodeLength     = 6
	CodeTTL        = 5 * time.Minute
	ResendCooldown = 60 * time.Second
	MaxAttempts    = 5

	AccessTTL          = 30 * time.Minute
	RefreshTTL         = 72 * time.Hour
	RememberRefreshTTL = 30 * 24 * time.Hour
)

var (
	ErrCodeExpired     = errors.New("인증코드가 만료됐어요. 다시 받아 주세요")
	ErrCodeInvalid     = errors.New("인증코드가 올바르지 않아요")
	ErrTooManyAttempts = errors.New("인증코드를 너무 많이 틀렸어요. 다시 받아 주세요")
)

// RetryAfterError는 인증코드를 너무 자주 요청했을 때 다시 시도할 수 있기까지의 시간을 담는다.
type RetryAfterError struct{ After time.Duration }

func (e *RetryAfterError) Error() string { return "잠시 후 다시 요청해 주세요" }

// EmailCode는 이메일로 보낸 인증코드(해시)와 확인 상태다.
type EmailCode struct {
	Email     string
	CodeHash  string
	Attempts  int
	SentAt    time.Time
	ExpiresAt time.Time
}

// NewCode는 6자리 숫자 인증코드를 만든다.
func NewCode() (string, error) {
	out := make([]byte, CodeLength)
	ten := big.NewInt(10)
	for i := range out {
		n, err := rand.Int(rand.Reader, ten)
		if err != nil {
			return "", err
		}
		out[i] = byte('0') + byte(n.Int64())
	}
	return string(out), nil
}

// HashCode는 저장용으로 코드를 해시한다. 이메일을 함께 묶어 다른 이메일의 코드와 섞이지 않게 한다.
func HashCode(secret, email, code string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(email))
	m.Write([]byte{0})
	m.Write([]byte(code))
	return hex.EncodeToString(m.Sum(nil))
}

// CanResend는 같은 이메일로 코드를 다시 보낼 수 있는지 알려 준다.
// 보낼 수 없으면 얼마 뒤에 가능한지를 함께 돌려준다.
func (c EmailCode) CanResend(now time.Time) (time.Duration, bool) {
	wait := c.SentAt.Add(ResendCooldown).Sub(now)
	if wait > 0 {
		return wait, false
	}
	return 0, true
}

// Check는 입력한 코드의 해시가 맞는지 확인한다. 시도 횟수를 올리는 일은 호출하는 쪽이 한다.
// attempts는 이번 시도를 포함하지 않은 지금까지의 틀린 횟수다.
func (c EmailCode) Check(now time.Time, hash string) error {
	if !now.Before(c.ExpiresAt) {
		return ErrCodeExpired
	}
	if c.Attempts >= MaxAttempts {
		return ErrTooManyAttempts
	}
	if subtle.ConstantTimeCompare([]byte(c.CodeHash), []byte(hash)) != 1 {
		return ErrCodeInvalid
	}
	return nil
}

// Session은 로그인 세션이다. refresh 토큰이 이 세션을 가리킨다.
type Session struct {
	ID             string
	UserID         string
	RememberDevice bool
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// RefreshTTLFor는 "기기 기억" 여부에 따른 refresh 토큰 수명이다.
func RefreshTTLFor(rememberDevice bool) time.Duration {
	if rememberDevice {
		return RememberRefreshTTL
	}
	return RefreshTTL
}

// IsActive는 세션이 아직 유효한지 알려 준다.
func (s Session) IsActive(now time.Time) bool { return now.Before(s.ExpiresAt) }

// Tokens는 발급한 토큰 한 쌍이다.
type Tokens struct {
	Access           string
	Refresh          string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}
