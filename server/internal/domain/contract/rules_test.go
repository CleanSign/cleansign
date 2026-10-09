package contract

import (
	"testing"
	"time"
)

func TestClampInt32(t *testing.T) {
	cases := []struct {
		in   int64
		want int32
	}{
		{-5, 0},
		{0, 0},
		{123, 123},
		{2_147_483_647, 2_147_483_647},
		{2_147_483_648, 2_147_483_647},
		{9_000_000_000_000, 2_147_483_647},
	}
	for _, c := range cases {
		if got := ClampInt32(c.in); got != c.want {
			t.Errorf("ClampInt32(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestClampOptional(t *testing.T) {
	if ClampOptional(nil) != nil {
		t.Fatal("nil이면 nil이어야 한다")
	}
	n := int64(5_000_000_000)
	if got := ClampOptional(&n); got == nil || *got != 2_147_483_647 {
		t.Fatalf("상한으로 잘려야 한다: %v", got)
	}
}

func TestValidateUpload(t *testing.T) {
	cases := []struct {
		name, mime string
		size       int64
		want       error
	}{
		{"a.pdf", "application/pdf", 100, nil},
		{"A.PDF", "application/octet-stream", 100, nil}, // detected by extension
		{"a.txt", "application/pdf", 100, nil},          // detected by MIME
		{"a.txt", "text/plain", 100, ErrNotPDF},
		{"a.pdf", "application/pdf", 0, ErrBadSize},
		{"a.pdf", "application/pdf", MaxUploadBytes, nil},
		{"a.pdf", "application/pdf", MaxUploadBytes + 1, ErrBadSize},
		{"a.txt", "text/plain", 0, ErrNotPDF}, // the PDF check comes first
	}
	for _, c := range cases {
		if got := ValidateUpload(c.name, c.mime, c.size); got != c.want {
			t.Errorf("ValidateUpload(%q, %q, %d) = %v, want %v", c.name, c.mime, c.size, got, c.want)
		}
	}
}

func TestIsStale(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	c := Contract{Status: StatusPending, CreatedAt: now.Add(-StaleAfter)}
	if c.IsStale(now) {
		t.Error("정확히 15분은 아직 stale이 아니다")
	}
	c.CreatedAt = now.Add(-StaleAfter - time.Second)
	if !c.IsStale(now) {
		t.Error("15분을 넘으면 stale이다")
	}
	c.Status = StatusCompleted
	if c.IsStale(now) {
		t.Error("pending이 아니면 stale이 아니다")
	}
}

func TestNormalizePage(t *testing.T) {
	cases := []struct {
		page, limit         string
		wantPage, wantLimit int
	}{
		{"", "", 1, 20},
		{"3", "10", 3, 10},
		{"0", "0", 1, 20},
		{"abc", "abc", 1, 20},
		{"-4", "-4", 1, 1},   // negative is raised to 1
		{"2.9", "7.9", 2, 7}, // floor
		{"1", "51", 1, 50},   // upper limit
		{"1", "1000", 1, 50},
		{" 2 ", " 5 ", 2, 5},
		{"NaN", "Inf", 1, 20},
	}
	for _, c := range cases {
		p, l := NormalizePage(c.page, c.limit)
		if p != c.wantPage || l != c.wantLimit {
			t.Errorf("NormalizePage(%q, %q) = (%d, %d), want (%d, %d)", c.page, c.limit, p, l, c.wantPage, c.wantLimit)
		}
	}
}
