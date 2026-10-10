package server

import (
	"strings"
	"testing"

	"github.com/CleanSign/cleansign/server/api"
)

func msgOf(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	status, msg := classify(err)
	if status != 400 {
		t.Fatalf("400이어야 한다: %d %v", status, err)
	}
	return msg
}

func TestValidateSendEmailCode(t *testing.T) {
	cases := []struct {
		name  string
		email string
		want  string // "" = 통과
	}{
		{"정상", "a@b.com", ""},
		{"앞뒤 공백은 domain이 정리하므로 통과", "  A@B.com ", ""},
		{"비어 있음", "", "이메일 형식이 올바르지 않아요"},
		{"254자 초과", strings.Repeat("a", 250) + "@b.com", "이메일 형식이 올바르지 않아요"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := msgOf(t, validateBody(api.SendEmailCodeRequest{Email: c.email})); got != c.want {
				t.Errorf("메시지 %q, 기대 %q", got, c.want)
			}
		})
	}
}

func TestValidateConfirmEmailCode(t *testing.T) {
	cases := []struct {
		name, email, code string
		want              string
	}{
		{"정상", "a@b.com", "123456", ""},
		{"코드 5자리", "a@b.com", "12345", "인증코드가 올바르지 않아요"},
		{"코드 7자리", "a@b.com", "1234567", "인증코드가 올바르지 않아요"},
		{"코드에 문자", "a@b.com", "12ab56", "인증코드가 올바르지 않아요"},
		{"코드 없음", "a@b.com", "", "인증코드가 올바르지 않아요"},
		{"이메일 없음", "", "123456", "이메일 형식이 올바르지 않아요"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := msgOf(t, validateBody(api.ConfirmEmailCodeRequest{Email: c.email, Code: c.code}))
			if got != c.want {
				t.Errorf("메시지 %q, 기대 %q", got, c.want)
			}
		})
	}
}

// 메시지 표에 없는 필드는 어느 필드인지 JSON 이름으로 알려 준다.
func TestValidateFallbackMessageUsesJSONName(t *testing.T) {
	type unlisted struct {
		Title string `json:"title" validate:"required"`
	}
	if got := msgOf(t, validateBody(unlisted{})); got != "title 값이 올바르지 않아요" {
		t.Errorf("대체 메시지: %q", got)
	}
}
