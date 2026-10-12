// Package auth contains the pure rules for email verification codes and sessions.
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
	// A single email may receive at most MaxSendsPerWindow codes per SendWindow (on top of ResendCooldown).
	MaxSendsPerWindow = 5
	SendWindow        = time.Hour

	AccessTTL          = 30 * time.Minute
	RefreshTTL         = 72 * time.Hour
	RememberRefreshTTL = 30 * 24 * time.Hour
)

var (
	ErrCodeExpired     = errors.New("인증코드가 만료됐어요. 다시 받아 주세요")
	ErrCodeInvalid     = errors.New("인증코드가 올바르지 않아요")
	ErrTooManyAttempts = errors.New("인증코드를 너무 많이 틀렸어요. 다시 받아 주세요")
	// ErrMailSend: the mail provider failed. The code row is deleted, so the user can retry right away.
	ErrMailSend = errors.New("인증코드 메일을 보내지 못했어요. 잠시 후 다시 시도해 주세요")
)

// RetryAfterError holds how long to wait before retrying when a verification code is requested too often.
type RetryAfterError struct{ After time.Duration }

func (e *RetryAfterError) Error() string { return "잠시 후 다시 요청해 주세요" }

// EmailCode is a verification code (hash) sent by email, along with its confirmation state.
type EmailCode struct {
	Email     string
	CodeHash  string
	Attempts  int
	SentAt    time.Time
	ExpiresAt time.Time
	// SendCount is how many codes were sent in the window that started at WindowStartedAt.
	SendCount       int
	WindowStartedAt time.Time
}

// NewCode creates a 6-digit numeric verification code.
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

// HashCode hashes a code for storage. It binds the email so codes don't mix across emails.
func HashCode(secret, email, code string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(email))
	m.Write([]byte{0})
	m.Write([]byte(code))
	return hex.EncodeToString(m.Sum(nil))
}

// CanResend reports whether a code can be sent again to the same email: the cooldown must have passed
// and the window limit must not be reached. If not, it also returns how long until it is possible.
func (c EmailCode) CanResend(now time.Time) (time.Duration, bool) {
	if wait := c.SentAt.Add(ResendCooldown).Sub(now); wait > 0 {
		return wait, false
	}
	if c.SendCount >= MaxSendsPerWindow {
		if wait := c.WindowStartedAt.Add(SendWindow).Sub(now); wait > 0 {
			return wait, false
		}
	}
	return 0, true
}

// NextSend returns the send counters for a code sent at now: the current window continues, or a new one starts
// when there is none yet or it has ended.
func (c EmailCode) NextSend(now time.Time) (sendCount int, windowStartedAt time.Time) {
	if c.WindowStartedAt.IsZero() || !now.Before(c.WindowStartedAt.Add(SendWindow)) {
		return 1, now
	}
	return c.SendCount + 1, c.WindowStartedAt
}

// Check verifies that the hash of the entered code matches. The caller is responsible for incrementing the attempt count.
// attempts is the number of wrong attempts so far, not including this one.
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

// Session is a login session. The refresh token points to this session.
type Session struct {
	ID             string
	UserID         string
	RememberDevice bool
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// RefreshTTLFor returns the refresh token lifetime depending on the "remember device" option.
func RefreshTTLFor(rememberDevice bool) time.Duration {
	if rememberDevice {
		return RememberRefreshTTL
	}
	return RefreshTTL
}

// IsActive reports whether the session is still valid.
func (s Session) IsActive(now time.Time) bool { return now.Before(s.ExpiresAt) }

// Tokens is a pair of issued tokens.
type Tokens struct {
	Access           string
	Refresh          string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}
