package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	otpapp "otp-pair-code-interview/internal/application/otp"
	coreotp "otp-pair-code-interview/internal/core/otp"
)

type Handler struct {
	service OTPService
	logger  zerolog.Logger
}

type OTPService interface {
	RequestOTP(ctx context.Context, userID string) (otpapp.RequestResult, error)
	ValidateOTP(ctx context.Context, userID, code string) (otpapp.ValidateResult, error)
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
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /otp/request [post]
func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req RequestOTPRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}

	result, err := h.service.RequestOTP(r.Context(), req.UserID)
	if err != nil {
		h.logOTPEvent(r, "otp_request_failed", req.UserID, err)
		writeDomainError(w, err)
		return
	}

	h.logOTPEvent(r, "otp_created", result.UserID, nil)
	writeJSON(w, http.StatusOK, RequestOTPResponse{UserID: result.UserID, OTP: result.OTP})
}

// ValidateOTP godoc
// @Summary Validate OTP
// @Description Validate an OTP for a user.
// @Tags otp
// @Accept json
// @Produce json
// @Param request body ValidateOTPRequest true "Validate OTP payload"
// @Success 200 {object} ValidateOTPResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 410 {object} ErrorResponse
// @Failure 429 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /otp/validate [post]
func (h *Handler) ValidateOTP(w http.ResponseWriter, r *http.Request) {
	var req ValidateOTPRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	if strings.TrimSpace(req.UserID) == "" || strings.TrimSpace(req.OTP) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "user_id and otp are required")
		return
	}

	result, err := h.service.ValidateOTP(r.Context(), req.UserID, req.OTP)
	if err != nil {
		h.logOTPEvent(r, "otp_validation_failed", req.UserID, err)
		writeDomainError(w, err)
		return
	}

	h.logOTPEvent(r, "otp_validated", result.UserID, nil)
	writeJSON(w, http.StatusOK, ValidateOTPResponse{UserID: result.UserID, Message: "OTP validated successfully."})
}

// Healthz godoc
// @Summary Health check
// @Description Check whether the service process is running.
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
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
		code, _, description := domainErrorDetails(err)
		log.Str("error_code", code).Str("error_description", description).Msg("otp event")
		return
	}
	log.Msg("otp event")
}

func decodeJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func writeDomainError(w http.ResponseWriter, err error) {
	code, status, description := domainErrorDetails(err)
	writeError(w, status, code, description)
}

func domainErrorDetails(err error) (string, int, string) {
	switch {
	case errors.Is(err, coreotp.ErrInvalidUserID), errors.Is(err, coreotp.ErrInvalidOTPInput):
		return "invalid_request", http.StatusBadRequest, "Request is invalid"
	case errors.Is(err, coreotp.ErrAlreadyActive):
		return "otp_already_active", http.StatusConflict, "An active OTP already exists"
	case errors.Is(err, coreotp.ErrNotFound):
		return "otp_not_found", http.StatusNotFound, "OTP not found"
	case errors.Is(err, coreotp.ErrExpired):
		return "otp_expired", http.StatusGone, "OTP has expired"
	case errors.Is(err, coreotp.ErrInvalidCode):
		return "otp_invalid", http.StatusBadRequest, "OTP is invalid"
	case errors.Is(err, coreotp.ErrBlocked):
		return "otp_blocked", http.StatusTooManyRequests, "OTP is blocked"
	case errors.Is(err, coreotp.ErrGenerationFailed):
		return "internal_error", http.StatusInternalServerError, "Failed to generate OTP"
	default:
		return "internal_error", http.StatusInternalServerError, "Internal server error"
	}
}

func writeError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, ErrorResponse{Error: code, ErrorDescription: description})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
