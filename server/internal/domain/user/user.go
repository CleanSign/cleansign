// Package user는 사용자와 약관 동의의 순수한 데이터와 규칙을 담는다.
package user

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

type User struct {
	ID            string
	Email         string
	Name          string
	EmailVerified bool
	CreatedAt     time.Time
}

// Consent는 가입 시 받은 약관 동의다.
type Consent struct {
	ID        string
	UserID    string
	Terms     bool
	Privacy   bool
	Marketing bool
	CreatedAt time.Time
}

var (
	ErrInvalidEmail    = errors.New("이메일 형식이 올바르지 않아요")
	ErrConsentRequired = errors.New("이용약관과 개인정보처리방침 동의는 필수입니다")
)

const maxEmailLen = 254

// NormalizeEmail은 공백을 없애고 소문자로 바꾼 뒤 형식을 확인한다.
func NormalizeEmail(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || len(s) > maxEmailLen {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(s)
	// "이름 <a@b.c>" 같은 형태를 막기 위해 입력과 주소가 같아야 한다
	if err != nil || addr.Address != s {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndex(s, "@")
	if at <= 0 || !strings.Contains(s[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return s, nil
}

// ValidateConsent는 필수 동의(이용약관, 개인정보처리방침)를 모두 받았는지 확인한다.
func ValidateConsent(terms, privacy bool) error {
	if !terms || !privacy {
		return ErrConsentRequired
	}
	return nil
}
