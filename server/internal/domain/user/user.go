// Package user contains the pure data and rules for users and terms consent.
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

// Consent is the terms consent given at sign-up.
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

// NormalizeEmail trims whitespace, lowercases, and then validates the format.
func NormalizeEmail(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || len(s) > maxEmailLen {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(s)
	// The input must equal the parsed address to reject forms like "Name <a@b.c>"
	if err != nil || addr.Address != s {
		return "", ErrInvalidEmail
	}
	at := strings.LastIndex(s, "@")
	if at <= 0 || !strings.Contains(s[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return s, nil
}

// ValidateConsent checks that all required consents (terms of service, privacy policy) were given.
func ValidateConsent(terms, privacy bool) error {
	if !terms || !privacy {
		return ErrConsentRequired
	}
	return nil
}
