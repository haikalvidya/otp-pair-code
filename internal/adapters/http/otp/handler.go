package otphttp

import (
	"context"
	"net/http"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	httpadapter "otp-pair-code-interview/internal/adapters/http"
	otpapp "otp-pair-code-interview/internal/application/otp"
)

type OTPService interface {
	RequestOTP(ctx context.Context, userID string) (otpapp.RequestResult, error)
	ValidateOTP(ctx context.Context, userID, code string) (otpapp.ValidateResult, error)
}

type Handler struct {
	service OTPService
	logger  zerolog.Logger
}

func NewHandler(service OTPService, logger zerolog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RequestOTP godoc
// @Summary Request OTP
// @Description Generate a new OTP for a user based on the configured active-OTP policy.
// @Tags otp
// @Accept json
// @Produce json
// @Param request body RequestOTPRequest true "Request OTP payload"
// @Success 200 {object} RequestOTPResponse
// @Failure 400 {object} httpadapter.ErrorResponse
// @Failure 409 {object} httpadapter.ErrorResponse
// @Failure 500 {object} httpadapter.ErrorResponse
// @Router /otp/request [post]
func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req RequestOTPRequest
	if err := httpadapter.DecodeJSON(r, &req); err != nil {
		httpadapter.WriteAPIError(w, r, httpadapter.APIError{StatusCode: http.StatusBadRequest, Code: "invalid_request", Message: "Request body is invalid"})
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		httpadapter.WriteAPIError(w, r, httpadapter.APIError{StatusCode: http.StatusBadRequest, Code: "invalid_request", Message: "user_id is required", Details: map[string]string{"user_id": "is required"}})
		return
	}

	result, err := h.service.RequestOTP(r.Context(), req.UserID)
	if err != nil {
		h.logOTPEvent(r, "otp_request_failed", req.UserID, err)
		httpadapter.WriteDomainError(w, r, err)
		return
	}

	h.logOTPEvent(r, "otp_created", result.UserID, nil)
	httpadapter.WriteSuccess(w, r, http.StatusOK, RequestOTPData{UserID: result.UserID, OTP: result.OTP})
}

// ValidateOTP godoc
// @Summary Validate OTP
// @Description Validate an OTP for a user.
// @Tags otp
// @Accept json
// @Produce json
// @Param request body ValidateOTPRequest true "Validate OTP payload"
// @Success 200 {object} ValidateOTPResponse
// @Failure 400 {object} httpadapter.ErrorResponse
// @Failure 404 {object} httpadapter.ErrorResponse
// @Failure 410 {object} httpadapter.ErrorResponse
// @Failure 429 {object} httpadapter.ErrorResponse
// @Failure 500 {object} httpadapter.ErrorResponse
// @Router /otp/validate [post]
func (h *Handler) ValidateOTP(w http.ResponseWriter, r *http.Request) {
	var req ValidateOTPRequest
	if err := httpadapter.DecodeJSON(r, &req); err != nil {
		httpadapter.WriteAPIError(w, r, httpadapter.APIError{StatusCode: http.StatusBadRequest, Code: "invalid_request", Message: "Request body is invalid"})
		return
	}
	if strings.TrimSpace(req.UserID) == "" || strings.TrimSpace(req.OTP) == "" {
		httpadapter.WriteAPIError(w, r, httpadapter.APIError{StatusCode: http.StatusBadRequest, Code: "invalid_request", Message: "user_id and otp are required", Details: map[string]string{"user_id": "is required", "otp": "is required"}})
		return
	}

	result, err := h.service.ValidateOTP(r.Context(), req.UserID, req.OTP)
	if err != nil {
		h.logOTPEvent(r, "otp_validation_failed", req.UserID, err)
		httpadapter.WriteDomainError(w, r, err)
		return
	}

	h.logOTPEvent(r, "otp_validated", result.UserID, nil)
	httpadapter.WriteSuccess(w, r, http.StatusOK, ValidateOTPData{UserID: result.UserID, Message: "OTP validated successfully."})
}

func (h *Handler) logOTPEvent(r *http.Request, event, userID string, err error) {
	log := h.logger.Info()
	if err != nil {
		log = h.logger.Warn()
	}
	log.Str("event", event).
		Str("request_id", chimiddleware.GetReqID(r.Context())).
		Str("user_id", userID)
	if err != nil {
		apiErr := httpadapter.MapDomainError(err)
		log.Str("error_code", apiErr.Code).Str("error_message", apiErr.Message).Msg("otp event")
		return
	}
	log.Msg("otp event")
}
