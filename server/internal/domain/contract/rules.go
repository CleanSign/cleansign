package contract

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	// MaxUploadBytes는 업로드할 수 있는 PDF의 최대 크기다.
	MaxUploadBytes = 10 * 1024 * 1024

	// StaleAfter가 지나도 pending이면 분석 작업이 사라진 것으로 본다.
	StaleAfter = 15 * time.Minute

	// StaleMessage는 사라진 분석을 failed로 바꿀 때 사용자에게 보여 줄 메시지다.
	StaleMessage = "분석이 중단됐어요. 다시 올려 주세요"

	DefaultPageSize = 20
	MaxPageSize     = 50

	int32Max = 2_147_483_647 // integer 컬럼 상한
)

var (
	ErrFileRequired = errors.New("file이 필요합니다")
	ErrNotPDF       = errors.New("PDF 파일만 올릴 수 있어요")
	ErrBadSize      = errors.New("파일 크기는 10MB 이하여야 해요")
)

// ClampInt32는 값을 0 이상 int32 최대값 이하로 맞춘다.
func ClampInt32(n int64) int32 {
	if n < 0 {
		return 0
	}
	if n > int32Max {
		return int32Max
	}
	return int32(n)
}

// ClampOptional은 값이 없으면 nil을 돌려준다.
func ClampOptional(n *int64) *int32 {
	if n == nil {
		return nil
	}
	v := ClampInt32(*n)
	return &v
}

// ValidateUpload는 업로드 파일이 규칙에 맞는지 확인한다.
// PDF 판별: MIME이 application/pdf이거나 파일 이름이 .pdf로 끝나야 한다.
func ValidateUpload(fileName, contentType string, size int64) error {
	isPDF := contentType == "application/pdf" || strings.HasSuffix(strings.ToLower(fileName), ".pdf")
	if !isPDF {
		return ErrNotPDF
	}
	if size <= 0 || size > MaxUploadBytes {
		return ErrBadSize
	}
	return nil
}

// IsStale은 분석 작업이 사라진 것으로 볼 만큼 pending이 오래됐는지 알려 준다.
func (c Contract) IsStale(now time.Time) bool {
	return c.Status == StatusPending && now.Sub(c.CreatedAt) > StaleAfter
}

// NormalizePage는 쿼리로 받은 page, limit 문자열을 보정한다.
//   - page: 정수로 내림, 숫자가 아니거나 0이면 1, 음수는 1
//   - limit: 숫자가 아니거나 0이면 20, 50 초과는 50, 음수는 1
func NormalizePage(pageRaw, limitRaw string) (page, limit int) {
	page = int(math.Max(1, orDefault(floorNumber(pageRaw), 1)))
	limit = int(math.Min(MaxPageSize, math.Max(1, orDefault(floorNumber(limitRaw), DefaultPageSize))))
	return page, limit
}

// floorNumber는 문자열을 숫자로 읽어 내린다. 읽을 수 없으면 0을 돌려준다.
func floorNumber(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return math.Floor(f)
}

// orDefault는 0(없음·잘못된 값)이면 기본값을 쓴다.
func orDefault(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}
