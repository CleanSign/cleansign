package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/CleanSign/cleansign/server/api"
	"github.com/CleanSign/cleansign/server/internal/domain/contract"
)

// Cap for the whole upload request: the file cap plus slack for multipart boundaries and the like.
const maxUploadRequestBytes = contract.MaxUploadBytes + 1<<20

func (h *handler) CreateContract(c echo.Context) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}

	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxUploadRequestBytes)
	fh, err := c.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return contract.ErrBadSize
		}
		return contract.ErrFileRequired
	}
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	id, err := h.Contracts.Submit(c.Request().Context(), userID, fh.Filename, fh.Header.Get(echo.HeaderContentType), fh.Size, f)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, api.CreateContractResponse{Id: id, Status: api.Pending})
}

func (h *handler) ListContracts(c echo.Context, params api.ListContractsParams) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}
	// page/limit are not validated on purpose: correcting bad values to defaults is the rule (domain.NormalizePage). Only status is checked
	if err := validateBody(params); err != nil {
		return err
	}
	ctx := c.Request().Context()
	page, limit := contract.NormalizePage(deref(params.Page), deref(params.Limit))

	// Recent list on the home screen: only completed and in-progress, limit items at a time
	if params.Status != nil && *params.Status == api.ListContractsParamsStatus("active") {
		items, err := h.Contracts.Recent(ctx, userID, limit)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, api.ContractPage{Items: listItems(items), Page: 1, Total: len(items), HasMore: false})
	}

	p, err := h.Contracts.List(ctx, userID, page, limit)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.ContractPage{Items: listItems(p.Items), Page: p.Page, Total: p.Total, HasMore: p.HasMore})
}

func (h *handler) GetContract(c echo.Context, id string) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}
	found, clauses, err := h.Contracts.Get(c.Request().Context(), userID, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contractResponse(found, clauses))
}

func (h *handler) DeleteContract(c echo.Context, id string) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}
	if err := h.Contracts.Delete(c.Request().Context(), userID, id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *handler) GetClause(c echo.Context, id, clauseID string) error {
	userID, err := requireUser(c)
	if err != nil {
		return err
	}
	cl, err := h.Contracts.Clause(c.Request().Context(), userID, id, clauseID)
	if err != nil {
		return err
	}
	law := cl.LawReference
	if law == nil {
		law = []string{}
	}
	return c.JSON(http.StatusOK, api.ClauseDetail{
		Id: cl.ID, ContractId: id, ClauseType: cl.ClauseType, Original: cl.Original, LossAmount: int(cl.LossAmount),
		Severity: api.Severity(cl.Severity), Confidence: cl.Confidence, Reason: cl.Reason, LawReference: law, SafeText: cl.SafeText,
	})
}

// --- Response shapes ---

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func listItems(items []contract.Summary) []api.ContractListItem {
	out := make([]api.ContractListItem, len(items))
	for i, s := range items {
		item := api.ContractListItem{
			Id: s.ID, Status: api.ContractStatus(s.Status), FileName: s.FileName, CreatedAt: s.CreatedAt, LossAmount: int(s.LossAmount),
		}
		if s.Status == contract.StatusCompleted {
			n := int(s.ClauseCount)
			item.RiskCount = &n
		}
		if s.Status == contract.StatusFailed {
			item.Error = s.ErrorMessage
		}
		out[i] = item
	}
	return out
}

// contractResponse builds the detail response. pending/failed are returned in the same shape.
func contractResponse(c contract.Contract, clauses []contract.Clause) api.Contract {
	resp := api.Contract{
		Id: c.ID, Status: api.ContractStatus(c.Status), FileName: c.FileName, CreatedAt: c.CreatedAt,
		LossAmount: int(c.LossAmount), RiskScore: intPtr(c.RiskScore), DepositAmount: intPtr(c.DepositAmount),
		Warnings: c.Warnings, RiskItems: make([]api.RiskItem, len(clauses)),
	}
	if resp.Warnings == nil {
		resp.Warnings = []string{}
	}
	if c.Status == contract.StatusPending && c.Progress != nil {
		resp.Progress = &api.Progress{Stage: c.Progress.Stage, Done: c.Progress.Done, Total: c.Progress.Total}
	}
	if c.Status == contract.StatusFailed {
		resp.Error = c.ErrorMessage
	}
	if c.Grade != nil {
		g := api.Severity(*c.Grade)
		resp.Grade = &g
	}
	if c.LossMin != nil && c.LossMax != nil {
		resp.LossRange = &[]int{int(*c.LossMin), int(*c.LossMax)}
	}
	if len(c.RiskFlow) > 0 {
		flow := json.RawMessage(c.RiskFlow)
		resp.Flow = &flow
	}
	for i, cl := range clauses {
		item := api.RiskItem{
			Id: cl.ID, Original: cl.Original, ClauseType: cl.ClauseType, Confidence: cl.Confidence,
			Severity: api.Severity(cl.Severity), LossAmount: int(cl.LossAmount), Reason: cl.Reason, SafeText: cl.SafeText,
		}
		// unlike the clause detail API, emit the comma-joined string as is
		if joined := strings.Join(cl.LawReference, ", "); joined != "" {
			item.LawReference = &joined
		}
		resp.RiskItems[i] = item
	}
	return resp
}

func intPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
