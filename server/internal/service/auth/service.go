// Package auth는 이메일 인증코드 로그인과 세션의 흐름을 조립한다.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/CleanSign/cleansign/server/internal/domain/auth"
	"github.com/CleanSign/cleansign/server/internal/domain/user"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
	"github.com/CleanSign/cleansign/server/internal/shared/errs"
	"github.com/CleanSign/cleansign/server/internal/shared/ids"
)

type Service struct {
	codes      CodeRepository
	users      UserStore
	sessions   SessionRepository
	mailer     Mailer
	tokens     TokenIssuer
	clock      clock.Clock
	codeSecret string
	log        *slog.Logger
}

func NewService(codes CodeRepository, users UserStore, sessions SessionRepository, mailer Mailer, tokens TokenIssuer, clk clock.Clock, codeSecret string, log *slog.Logger) *Service {
	return &Service{codes: codes, users: users, sessions: sessions, mailer: mailer, tokens: tokens, clock: clk, codeSecret: codeSecret, log: log}
}

// SendCode는 이메일로 6자리 인증코드를 보낸다. 다시 보내려면 쿨다운이 지나야 한다.
// 돌려주는 값은 다음 요청까지 기다려야 하는 시간이다.
func (s *Service) SendCode(ctx context.Context, rawEmail string) (time.Duration, error) {
	email, err := user.NormalizeEmail(rawEmail)
	if err != nil {
		return 0, err
	}
	now := s.clock.Now()

	existing, err := s.codes.Get(ctx, email)
	switch {
	case err == nil:
		if wait, ok := existing.CanResend(now); !ok {
			return 0, &auth.RetryAfterError{After: wait}
		}
	case !errors.Is(err, errs.ErrNotFound):
		return 0, err
	}

	code, err := auth.NewCode()
	if err != nil {
		return 0, err
	}
	if err := s.codes.Upsert(ctx, auth.EmailCode{
		Email:     email,
		CodeHash:  auth.HashCode(s.codeSecret, email, code),
		SentAt:    now,
		ExpiresAt: now.Add(auth.CodeTTL),
	}); err != nil {
		return 0, err
	}
	if err := s.mailer.SendCode(ctx, email, code, auth.CodeTTL); err != nil {
		// 보내지 못했으면 기다리지 않고 바로 다시 요청할 수 있게 지운다
		if delErr := s.codes.Delete(ctx, email); delErr != nil {
			s.log.Warn("failed to delete unsent code", "error", delErr)
		}
		return 0, fmt.Errorf("send code mail: %w", err)
	}
	return auth.ResendCooldown, nil
}

// ConfirmResult는 로그인에 성공했을 때의 결과다.
type ConfirmResult struct {
	User   user.User
	Tokens auth.Tokens
}

// Confirm은 인증코드를 확인하고 로그인한다. 처음 보는 이메일이면 가입시킨다.
func (s *Service) Confirm(ctx context.Context, rawEmail, code string, rememberDevice bool) (ConfirmResult, error) {
	email, err := user.NormalizeEmail(rawEmail)
	if err != nil {
		return ConfirmResult{}, err
	}
	now := s.clock.Now()

	stored, err := s.codes.Get(ctx, email)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return ConfirmResult{}, auth.ErrCodeExpired
		}
		return ConfirmResult{}, err
	}
	if err := stored.Check(now, auth.HashCode(s.codeSecret, email, code)); err != nil {
		switch {
		case errors.Is(err, auth.ErrCodeInvalid):
			if incErr := s.codes.IncrementAttempts(ctx, email); incErr != nil {
				s.log.Warn("failed to count attempt", "error", incErr)
			}
		case errors.Is(err, auth.ErrCodeExpired):
			if delErr := s.codes.Delete(ctx, email); delErr != nil {
				s.log.Warn("failed to delete expired code", "error", delErr)
			}
		}
		return ConfirmResult{}, err
	}
	if err := s.codes.Delete(ctx, email); err != nil {
		return ConfirmResult{}, err
	}

	u, err := s.loginOrCreate(ctx, email, now)
	if err != nil {
		return ConfirmResult{}, err
	}

	session := auth.Session{
		ID:             ids.New(),
		UserID:         u.ID,
		RememberDevice: rememberDevice,
		ExpiresAt:      now.Add(auth.RefreshTTLFor(rememberDevice)),
		CreatedAt:      now,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return ConfirmResult{}, err
	}
	tokens, err := s.issue(u.ID, session, now)
	if err != nil {
		return ConfirmResult{}, err
	}
	return ConfirmResult{User: u, Tokens: tokens}, nil
}

func (s *Service) loginOrCreate(ctx context.Context, email string, now time.Time) (user.User, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, errs.ErrNotFound) {
		return user.User{}, err
	}
	created, err := s.users.Create(ctx, user.User{
		ID:            ids.New(),
		Email:         email,
		EmailVerified: true, // 인증코드로 이메일을 확인했다
		CreatedAt:     now,
	})
	if err != nil {
		// 같은 이메일로 동시에 가입한 경우: 먼저 만들어진 사용자를 쓴다
		if again, getErr := s.users.GetByEmail(ctx, email); getErr == nil {
			return again, nil
		}
		return user.User{}, err
	}
	return created, nil
}

func (s *Service) issue(userID string, session auth.Session, now time.Time) (auth.Tokens, error) {
	access, accessExp, err := s.tokens.Access(userID, session.ID, now)
	if err != nil {
		return auth.Tokens{}, err
	}
	refresh, err := s.tokens.Refresh(userID, session.ID, session.ExpiresAt)
	if err != nil {
		return auth.Tokens{}, err
	}
	return auth.Tokens{Access: access, Refresh: refresh, AccessExpiresAt: accessExp, RefreshExpiresAt: session.ExpiresAt}, nil
}

// Refresh는 refresh 토큰으로 새 access 토큰을 발급한다. refresh 토큰 자체는 바뀌지 않는다.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (auth.Tokens, error) {
	claims, err := s.tokens.Parse(refreshToken, false)
	if err != nil || claims.Kind != KindRefresh {
		return auth.Tokens{}, errs.ErrUnauthorized
	}
	session, err := s.activeSession(ctx, claims)
	if err != nil {
		return auth.Tokens{}, err
	}
	access, accessExp, err := s.tokens.Access(claims.UserID, session.ID, s.clock.Now())
	if err != nil {
		return auth.Tokens{}, err
	}
	return auth.Tokens{Access: access, Refresh: refreshToken, AccessExpiresAt: accessExp, RefreshExpiresAt: session.ExpiresAt}, nil
}

// Authenticate는 access 토큰이 가리키는 사용자와 세션을 확인한다. 로그아웃된 세션은 거절한다.
func (s *Service) Authenticate(ctx context.Context, accessToken string) (userID, sessionID string, err error) {
	claims, err := s.tokens.Parse(accessToken, false)
	if err != nil || claims.Kind != KindAccess {
		return "", "", errs.ErrUnauthorized
	}
	session, err := s.activeSession(ctx, claims)
	if err != nil {
		return "", "", err
	}
	return claims.UserID, session.ID, nil
}

func (s *Service) activeSession(ctx context.Context, claims Claims) (auth.Session, error) {
	session, err := s.sessions.Get(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return auth.Session{}, errs.ErrUnauthorized
		}
		return auth.Session{}, err
	}
	if session.UserID != claims.UserID || !session.IsActive(s.clock.Now()) {
		return auth.Session{}, errs.ErrUnauthorized
	}
	return session, nil
}

// Logout은 토큰이 가리키는 세션을 폐기한다. 만료된 토큰이어도 서명이 맞으면 폐기한다.
// 이미 폐기됐거나 토큰이 없어도 오류가 아니다.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	claims, err := s.tokens.Parse(token, true)
	if err != nil {
		return nil
	}
	return s.sessions.Delete(ctx, claims.SessionID)
}

// RevokeAll은 사용자의 모든 세션을 폐기한다.
func (s *Service) RevokeAll(ctx context.Context, userID string) error {
	return s.sessions.DeleteByUser(ctx, userID)
}

// PurgeExpired는 만료된 인증코드와 세션을 지운다.
func (s *Service) PurgeExpired(ctx context.Context) error {
	now := s.clock.Now()
	codes, err := s.codes.DeleteExpired(ctx, now)
	if err != nil {
		return err
	}
	sessions, err := s.sessions.DeleteExpired(ctx, now)
	if err != nil {
		return err
	}
	if codes+sessions > 0 {
		s.log.Info("purged expired auth data", "codes", codes, "sessions", sessions)
	}
	return nil
}
