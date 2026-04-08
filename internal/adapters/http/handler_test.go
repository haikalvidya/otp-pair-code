package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	otpapp "otp-pair-code-interview/internal/application/otp"
	coreotp "otp-pair-code-interview/internal/core/otp"
)

type fakeService struct {
	requestResult  otpapp.RequestResult
	requestErr     error
	validateResult otpapp.ValidateResult
	validateErr    error
}

func (f fakeService) RequestOTP(context.Context, string) (otpapp.RequestResult, error) {
	return f.requestResult, f.requestErr
}

func (f fakeService) ValidateOTP(context.Context, string, string) (otpapp.ValidateResult, error) {
	return f.validateResult, f.validateErr
}

func TestRequestRoute(t *testing.T) {
	handler := NewHandler(fakeService{requestResult: otpapp.RequestResult{UserID: "Robert", OTP: "12345"}}, zerolog.Nop())
	router := NewRouter(handler, zerolog.Nop(), 5*time.Second)

	body, _ := json.Marshal(RequestOTPRequest{UserID: "Robert"})
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestValidateRoute(t *testing.T) {
	handler := NewHandler(fakeService{validateErr: coreotp.ErrInvalidCode}, zerolog.Nop())
	router := NewRouter(handler, zerolog.Nop(), 5*time.Second)

	body, _ := json.Marshal(ValidateOTPRequest{UserID: "Robert", OTP: "99999"})
	req := httptest.NewRequest(http.MethodPost, "/otp/validate", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestHealthRoute(t *testing.T) {
	handler := NewHandler(fakeService{}, zerolog.Nop())
	router := NewRouter(handler, zerolog.Nop(), 5*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestSwaggerRoute(t *testing.T) {
	handler := NewHandler(fakeService{}, zerolog.Nop())
	router := NewRouter(handler, zerolog.Nop(), 5*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code == http.StatusNotFound {
		t.Fatalf("expected swagger route to be mounted")
	}
}

func TestDomainErrorDetails(t *testing.T) {
	code, status, _ := domainErrorDetails(coreotp.ErrAlreadyActive)
	if code != "otp_already_active" || status != http.StatusConflict {
		t.Fatalf("unexpected mapping: %s %d", code, status)
	}

	code, status, _ = domainErrorDetails(errors.New("boom"))
	if code != "internal_error" || status != http.StatusInternalServerError {
		t.Fatalf("unexpected default mapping: %s %d", code, status)
	}
}
