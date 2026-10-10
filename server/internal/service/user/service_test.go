package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	domain "github.com/CleanSign/cleansign/server/internal/domain/user"
	"github.com/CleanSign/cleansign/server/internal/shared/clock"
)

func TestRecordConsentRejectsBeforeTouchingTheStore(t *testing.T) {
	repo := NewMockRepository(gomock.NewController(t)) // 기대가 없으므로 저장소가 불리면 실패한다
	svc := NewService(repo, &clock.Fixed{T: time.Now()})

	for _, c := range [][2]bool{{false, true}, {true, false}, {false, false}} {
		if _, err := svc.RecordConsent(context.Background(), "u1", c[0], c[1], false); !errors.Is(err, domain.ErrConsentRequired) {
			t.Errorf("RecordConsent(%v): %v", c, err)
		}
	}
}

func TestRecordConsentStoresWithTimestamp(t *testing.T) {
	now := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	repo := NewMockRepository(gomock.NewController(t))
	repo.EXPECT().CreateConsent(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c domain.Consent) (domain.Consent, error) {
			if c.UserID != "u1" || !c.Terms || !c.Privacy || !c.Marketing || !c.CreatedAt.Equal(now) || c.ID == "" {
				t.Errorf("저장할 동의: %+v", c)
			}
			return c, nil
		})

	svc := NewService(repo, &clock.Fixed{T: now})
	if _, err := svc.RecordConsent(context.Background(), "u1", true, true, true); err != nil {
		t.Fatal(err)
	}
}
