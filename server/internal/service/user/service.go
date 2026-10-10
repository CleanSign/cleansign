// Package user assembles the flow for user info and terms consent.
package user

import (
	"context"

	"github.com/CleanSign/cleansign/server/internal/domain/user"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
	"github.com/CleanSign/cleansign/server/internal/shared/ids"
)

//go:generate go tool mockgen -typed -source=service.go -destination=repository_mock_test.go -package=user -self_package=github.com/CleanSign/cleansign/server/internal/service/user

// Repository is the user store port. It returns errs.ErrNotFound if missing.
type Repository interface {
	GetByID(ctx context.Context, id string) (user.User, error)
	CreateConsent(ctx context.Context, c user.Consent) (user.Consent, error)
}

// 생성 패키지 이름을 임시로 다르게 주는 이유: domain/user와 이름이 같으면 생성기가 `user.` 접두사를 지워 버린다.
//
//go:generate go tool ifacemaker -f service.go -s service -i Service -p usergen -o service_interface.go -y "Service는 user 흐름의 공개 메서드다. handler가 이 인터페이스에 의존하고, 테스트에서는 mock으로 바꾼다."
//go:generate perl -pi -e "s/^package usergen/package user/" service_interface.go
type service struct {
	repo  Repository
	clock clock.Clock
}

func NewService(repo Repository, clk clock.Clock) Service {
	return &service{repo: repo, clock: clk}
}

// Me returns the logged-in user's info.
func (s *service) Me(ctx context.Context, userID string) (user.User, error) {
	return s.repo.GetByID(ctx, userID)
}

// RecordConsent saves the terms consent. The terms of service and privacy policy are required.
func (s *service) RecordConsent(ctx context.Context, userID string, terms, privacy, marketing bool) (user.Consent, error) {
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
