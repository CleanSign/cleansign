package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/CleanSign/cleansign/server/api"
)

func (h *handler) GetMe(c echo.Context) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}
	u, err := h.Users.Me(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.Me{Id: u.ID, Name: u.Name, Email: u.Email, EmailVerified: u.EmailVerified})
}

// CreateConsent는 약관 동의를 저장한다.
// 필드 타입 오류와 필수 동의 누락은 기존 API와 같은 메시지로 알려 주기 위해 본문을 직접 읽는다.
func (h *handler) CreateConsent(c echo.Context) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}

	var body map[string]any
	// 읽지 못하거나 객체가 아니면 필드가 없는 것으로 본다
	if err := json.NewDecoder(io.LimitReader(c.Request().Body, maxJSONBody)).Decode(&body); err != nil {
		body = nil
	}
	terms, termsOK := body["terms"].(bool)
	privacy, privacyOK := body["privacy"].(bool)
	if !termsOK || !privacyOK {
		return badRequest("terms, privacy는 boolean이어야 합니다")
	}
	marketing, _ := body["marketing"].(bool) // boolean이 아니면 false

	consent, err := h.Users.RecordConsent(c.Request().Context(), userID, terms, privacy, marketing)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, api.Consent{
		Id: consent.ID, UserId: consent.UserID, Terms: consent.Terms, Privacy: consent.Privacy,
		Marketing: consent.Marketing, CreatedAt: consent.CreatedAt,
	})
}
