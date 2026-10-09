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

// CreateConsent saves the terms consent.
// It reads the body directly so field type errors and missing required consents are reported with the same messages as the existing API.
func (h *handler) CreateConsent(c echo.Context) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}

	var body map[string]any
	// if it cannot be read or is not an object, treat the fields as absent
	if err := json.NewDecoder(io.LimitReader(c.Request().Body, maxJSONBody)).Decode(&body); err != nil {
		body = nil
	}
	terms, termsOK := body["terms"].(bool)
	privacy, privacyOK := body["privacy"].(bool)
	if !termsOK || !privacyOK {
		return badRequest("terms, privacy는 boolean이어야 합니다")
	}
	marketing, _ := body["marketing"].(bool) // false if not a boolean

	consent, err := h.Users.RecordConsent(c.Request().Context(), userID, terms, privacy, marketing)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, api.Consent{
		Id: consent.ID, UserId: consent.UserID, Terms: consent.Terms, Privacy: consent.Privacy,
		Marketing: consent.Marketing, CreatedAt: consent.CreatedAt,
	})
}
