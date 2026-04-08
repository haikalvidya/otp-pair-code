package httpadapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	httpadapter "otp-pair-code/internal/adapters/http"
	healthhttp "otp-pair-code/internal/adapters/http/health"
	otphttp "otp-pair-code/internal/adapters/http/otp"
	otpapp "otp-pair-code/internal/application/otp"
	coreotp "otp-pair-code/internal/core/otp"
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
	otpHandler := otphttp.NewHandler(fakeService{requestResult: otpapp.RequestResult{UserID: "Robert", OTP: "12345"}}, zerolog.Nop())
	router := httpadapter.NewRouter(otpHandler, healthhttp.NewHandler(), zerolog.Nop(), 5*time.Second)

	body, _ := json.Marshal(otphttp.RequestOTPRequest{UserID: "Robert"})
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var response otphttp.RequestOTPResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Data.UserID != "Robert" || response.Data.OTP != "12345" {
		t.Fatalf("unexpected response data: %+v", response.Data)
	}
	if response.Meta.RequestID == "" {
		t.Fatal("expected request_id in response meta")
	}
}

func TestValidateRoute(t *testing.T) {
	otpHandler := otphttp.NewHandler(fakeService{validateErr: coreotp.ErrInvalidCode}, zerolog.Nop())
	router := httpadapter.NewRouter(otpHandler, healthhttp.NewHandler(), zerolog.Nop(), 5*time.Second)

	body, _ := json.Marshal(otphttp.ValidateOTPRequest{UserID: "Robert", OTP: "99999"})
	req := httptest.NewRequest(http.MethodPost, "/otp/validate", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}

	var response httpadapter.ErrorResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if response.Error.Code != "otp_invalid" {
		t.Fatalf("expected otp_invalid, got %s", response.Error.Code)
	}
	if response.Meta.RequestID == "" {
		t.Fatal("expected request_id in error response meta")
	}
}

func TestHealthRoute(t *testing.T) {
	router := httpadapter.NewRouter(otphttp.NewHandler(fakeService{}, zerolog.Nop()), healthhttp.NewHandler(), zerolog.Nop(), 5*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var response healthhttp.HealthResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	if response.Data.Status != "ok" {
		t.Fatalf("expected health ok, got %+v", response.Data)
	}
}

func TestSwaggerRoute(t *testing.T) {
	router := httpadapter.NewRouter(otphttp.NewHandler(fakeService{}, zerolog.Nop()), healthhttp.NewHandler(), zerolog.Nop(), 5*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code == http.StatusNotFound {
		t.Fatalf("expected swagger route to be mounted")
	}
}

func TestMapDomainError(t *testing.T) {
	apiErr := httpadapter.MapDomainError(coreotp.ErrAlreadyActive)
	if apiErr.Code != "otp_already_active" || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("unexpected mapping: %+v", apiErr)
	}

	apiErr = httpadapter.MapDomainError(assertErr{})
	if apiErr.Code != "internal_error" || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected default mapping: %+v", apiErr)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "boom" }
