// Package user는 사용자 정보와 약관 동의의 흐름을 조립한다.
package user

import (
	"context"

	"github.com/CleanSign/cleansign/server/internal/domain/user"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
	"github.com/CleanSign/cleansign/server/internal/shared/ids"
)

// Repository는 사용자 저장소 포트다. 없으면 errs.ErrNotFound를 돌려준다.
type Repository interface {
	GetByID(ctx context.Context, id string) (user.User, error)
	CreateConsent(ctx context.Context, c user.Consent) (user.Consent, error)
}

type Service struct {
	repo  Repository
	clock clock.Clock
}

func NewService(repo Repository, clk clock.Clock) *Service {
	return &Service{repo: repo, clock: clk}
}

// Me는 로그인한 사용자의 정보를 돌려준다.
func (s *Service) Me(ctx context.Context, userID string) (user.User, error) {
	return s.repo.GetByID(ctx, userID)
}

// RecordConsent는 약관 동의를 저장한다. 이용약관과 개인정보처리방침은 필수다.
func (s *Service) RecordConsent(ctx context.Context, userID string, terms, privacy, marketing bool) (user.Consent, error) {
	if err := user.ValidateConsent(terms, privacy); err != nil {
		return user.Consent{}, err
	}
	return s.repo.CreateConsent(ctx, user.Consent{
		ID:        ids.New(),
		UserID:    userID,
		Terms:     terms,
		Privacy:   privacy,
		Marketing: marketing,
		CreatedAt: s.clock.Now(),
	})
}
