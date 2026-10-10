package server

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/CleanSign/cleansign/server/internal/domain/auth"
	"github.com/CleanSign/cleansign/server/internal/domain/user"
)

// 요청 본문의 "모양" 규칙은 openapi.yaml의 x-oapi-codegen-extra-tags(validate)에 적고,
// make gen이 생성 구조체의 태그로 옮긴다. 여기서는 그 태그를 읽어 검사한다.
// 형식·시간·상태처럼 의미가 들어간 규칙은 domain에 둔다(예: 이메일 형식, 인증코드 쿨다운).
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	// 오류에 구조체 필드 이름이 아니라 JSON 이름(email, code)이 나오게 한다
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return f.Name
		}
		return name
	})
	return v
}

// fieldMessages는 "구조체.필드"에서 규칙을 어겼을 때 보여 줄 메시지다.
// 옛 API와 같은 한국어 문구를 그대로 지키려고 규칙 이름이 아니라 필드 기준으로 정한다.
var fieldMessages = map[string]string{
	"SendEmailCodeRequest.Email":    user.ErrInvalidEmail.Error(),
	"ConfirmEmailCodeRequest.Email": user.ErrInvalidEmail.Error(),
	"ConfirmEmailCodeRequest.Code":  auth.ErrCodeInvalid.Error(),
	// terms·privacy는 true여야 한다(false면 필수 동의 누락). 타입이 boolean이 아닌 경우는 handler가 먼저 걸러낸다
	"ConsentRequest.Terms":   user.ErrConsentRequired.Error(),
	"ConsentRequest.Privacy": user.ErrConsentRequired.Error(),
}

// validateBody는 생성 구조체의 validate 태그를 검사한다. 위반하면 400과 한국어 메시지를 돌려준다.
// 첫 번째 위반만 알려 준다.
func validateBody(v any) error {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}
	var verr validator.ValidationErrors
	if !errors.As(err, &verr) || len(verr) == 0 {
		return err // 검증기 설정 오류 같은 예상 밖의 문제: 500
	}
	first := verr[0]
	if msg, ok := fieldMessages[first.StructNamespace()]; ok {
		return badRequest(msg)
	}
	// 표에 없는 필드: 어느 필드인지는 알려 준다
	return badRequest(fmt.Sprintf("%s 값이 올바르지 않아요", first.Field()))
}
