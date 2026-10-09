// Package token creates and verifies access/refresh tokens as HS256 JWTs.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/CleanSign/cleansign/server/internal/domain/auth"
	svc "github.com/CleanSign/cleansign/server/internal/service/auth"
)

const issuer = "cleansign"

var ErrInvalid = errors.New("invalid token")

type claims struct {
	SessionID string        `json:"sid"`
	Kind      svc.TokenKind `json:"typ"`
	jwt.RegisteredClaims
}

type Issuer struct{ secret []byte }

var _ svc.TokenIssuer = (*Issuer)(nil)

func New(secret string) *Issuer { return &Issuer{secret: []byte(secret)} }

func (i *Issuer) Access(userID, sessionID string, now time.Time) (string, time.Time, error) {
	exp := now.Add(auth.AccessTTL)
	tok, err := i.sign(userID, sessionID, svc.KindAccess, now, exp)
	return tok, exp, err
}

func (i *Issuer) Refresh(userID, sessionID string, expiresAt time.Time) (string, error) {
	return i.sign(userID, sessionID, svc.KindRefresh, time.Now(), expiresAt)
}

func (i *Issuer) sign(userID, sessionID string, kind svc.TokenKind, issuedAt, expiresAt time.Time) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		SessionID: sessionID,
		Kind:      kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})
	signed, err := t.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Parse verifies the signature and returns the token's values.
// If ignoreExpiry is true, only expiry is ignored (e.g. logout, where an expired token only needs to point to a session).
func (i *Issuer) Parse(token string, ignoreExpiry bool) (svc.Claims, error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	}
	if ignoreExpiry {
		opts = []jwt.ParserOption{
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(issuer),
			jwt.WithoutClaimsValidation(),
		}
	}
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return i.secret, nil }, opts...)
	if err != nil {
		return svc.Claims{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if ignoreExpiry && c.Issuer != issuer { // WithoutClaimsValidation also skips the issuer check
		return svc.Claims{}, ErrInvalid
	}
	if c.Subject == "" || c.SessionID == "" || (c.Kind != svc.KindAccess && c.Kind != svc.KindRefresh) {
		return svc.Claims{}, ErrInvalid
	}
	return svc.Claims{UserID: c.Subject, SessionID: c.SessionID, Kind: c.Kind}, nil
}
