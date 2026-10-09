package token

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	svc "github.com/CleanSign/cleansign/server/internal/service/auth"
)

const secret = "test-secret-at-least-16-chars"

func TestAccessAndRefreshRoundTrip(t *testing.T) {
	i := New(secret)
	now := time.Now()

	access, exp, err := i.Access("u1", "s1", now)
	if err != nil || exp.Before(now) {
		t.Fatalf("Access: %v %v", exp, err)
	}
	got, err := i.Parse(access, false)
	if err != nil || got.UserID != "u1" || got.SessionID != "s1" || got.Kind != svc.KindAccess {
		t.Fatalf("Parse(access) = %+v, %v", got, err)
	}

	refresh, err := i.Refresh("u1", "s1", now.Add(72*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := i.Parse(refresh, false); err != nil || got.Kind != svc.KindRefresh {
		t.Fatalf("Parse(refresh) = %+v, %v", got, err)
	}
}

func TestParseRejects(t *testing.T) {
	i := New(secret)
	now := time.Now()
	good, _, _ := i.Access("u1", "s1", now)

	if _, err := New("another-secret-at-least-16").Parse(good, false); !errors.Is(err, ErrInvalid) {
		t.Errorf("다른 비밀키로 서명한 토큰: %v", err)
	}
	if _, err := i.Parse(good[:len(good)-2]+"xx", false); !errors.Is(err, ErrInvalid) {
		t.Errorf("서명이 깨진 토큰: %v", err)
	}
	if _, err := i.Parse("garbage", false); !errors.Is(err, ErrInvalid) {
		t.Errorf("토큰이 아닌 문자열: %v", err)
	}

	// reject alg=none
	none := jwt.NewWithClaims(jwt.SigningMethodNone, claims{SessionID: "s1", Kind: svc.KindAccess, RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: "u1", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}})
	s, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := i.Parse(s, false); !errors.Is(err, ErrInvalid) {
		t.Errorf("alg=none: %v", err)
	}
	if _, err := i.Parse(s, true); !errors.Is(err, ErrInvalid) {
		t.Errorf("alg=none (만료 무시 모드): %v", err)
	}

	// reject a different issuer
	other := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{SessionID: "s1", Kind: svc.KindAccess, RegisteredClaims: jwt.RegisteredClaims{Issuer: "someone", Subject: "u1", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}})
	s, _ = other.SignedString([]byte(secret))
	for _, ignore := range []bool{false, true} {
		if _, err := i.Parse(s, ignore); !errors.Is(err, ErrInvalid) {
			t.Errorf("다른 발급자(ignoreExpiry=%v): %v", ignore, err)
		}
	}

	// reject if a required value is missing
	noSid := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{Kind: svc.KindAccess, RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: "u1", ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}})
	s, _ = noSid.SignedString([]byte(secret))
	if _, err := i.Parse(s, false); !errors.Is(err, ErrInvalid) {
		t.Errorf("sid 없음: %v", err)
	}
	if !strings.Contains(good, ".") {
		t.Fatal("JWT 형식이 아니다")
	}
}

func TestExpiry(t *testing.T) {
	i := New(secret)
	past := time.Now().Add(-2 * time.Hour)
	expired, _, err := i.Access("u1", "s1", past) // a 30-minute token that expired an hour and a half ago
	if err != nil {
		t.Fatal(err)
	}
	if _, err := i.Parse(expired, false); !errors.Is(err, ErrInvalid) {
		t.Errorf("만료된 토큰은 거절: %v", err)
	}
	got, err := i.Parse(expired, true)
	if err != nil || got.SessionID != "s1" {
		t.Errorf("만료를 무시하면 값을 읽을 수 있어야 한다: %+v %v", got, err)
	}
}
