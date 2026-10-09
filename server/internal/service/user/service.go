// Package user assembles the flow for user info and terms consent.
package user

import (
	"context"

	"github.com/CleanSign/cleansign/server/internal/domain/user"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
	"github.com/CleanSign/cleansign/server/internal/shared/ids"
)

// Repository is the user store port. It returns errs.ErrNotFound if missing.
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

// Me returns the logged-in user's info.
func (s *Service) Me(ctx context.Context, userID string) (user.User, error) {
	return s.repo.GetByID(ctx, userID)
}

// RecordConsent saves the terms consent. The terms of service and privacy policy are required.
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
