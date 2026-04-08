package otphttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	otpapp "otp-pair-code-interview/internal/application/otp"
	coreotp "otp-pair-code-interview/internal/core/otp"
)

type stubService struct {
	requestFunc  func(context.Context, string) (otpapp.RequestResult, error)
	validateFunc func(context.Context, string, string) (otpapp.ValidateResult, error)
}

func (s stubService) RequestOTP(ctx context.Context, userID string) (otpapp.RequestResult, error) {
	if s.requestFunc == nil {
		return otpapp.RequestResult{}, nil
	}
	return s.requestFunc(ctx, userID)
}

func (s stubService) ValidateOTP(ctx context.Context, userID, code string) (otpapp.ValidateResult, error) {
	if s.validateFunc == nil {
		return otpapp.ValidateResult{}, nil
	}
	return s.validateFunc(ctx, userID, code)
}

func TestRequestOTPRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(stubService{}, zerolog.Nop())
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewBufferString("{"))
	resp := httptest.NewRecorder()

	handler.RequestOTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errorBody := response["error"].(map[string]any)
	if errorBody["code"] != "invalid_request" {
		t.Fatalf("expected invalid_request, got %v", errorBody["code"])
	}
}

func TestRequestOTPRejectsMissingUserID(t *testing.T) {
	handler := NewHandler(stubService{}, zerolog.Nop())
	body, _ := json.Marshal(RequestOTPRequest{UserID: "  "})
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	handler.RequestOTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestRequestOTPMapsDomainError(t *testing.T) {
	handler := NewHandler(stubService{
		requestFunc: func(context.Context, string) (otpapp.RequestResult, error) {
			return otpapp.RequestResult{}, coreotp.ErrAlreadyActive
		},
	}, zerolog.Nop())
	body, _ := json.Marshal(RequestOTPRequest{UserID: "Robert"})
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	handler.RequestOTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", resp.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errorBody := response["error"].(map[string]any)
	if errorBody["code"] != coreotp.CodeAlreadyActive {
		t.Fatalf("expected %s, got %v", coreotp.CodeAlreadyActive, errorBody["code"])
	}
}

func TestRequestOTPSuccess(t *testing.T) {
	var capturedUserID string
	handler := NewHandler(stubService{
		requestFunc: func(_ context.Context, userID string) (otpapp.RequestResult, error) {
			capturedUserID = userID
			return otpapp.RequestResult{UserID: userID, OTP: "12345"}, nil
		},
	}, zerolog.Nop())
	body, _ := json.Marshal(RequestOTPRequest{UserID: "Robert"})
	req := httptest.NewRequest(http.MethodPost, "/otp/request", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	handler.RequestOTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if capturedUserID != "Robert" {
		t.Fatalf("expected service to receive Robert, got %s", capturedUserID)
	}

	var response RequestOTPResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Data.OTP != "12345" {
		t.Fatalf("expected OTP 12345, got %s", response.Data.OTP)
	}
}

func TestValidateOTPRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(stubService{}, zerolog.Nop())
	req := httptest.NewRequest(http.MethodPost, "/otp/validate", bytes.NewBufferString("{"))
	resp := httptest.NewRecorder()

	handler.ValidateOTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestValidateOTPRejectsMissingFields(t *testing.T) {
	handler := NewHandler(stubService{}, zerolog.Nop())
	body, _ := json.Marshal(ValidateOTPRequest{UserID: "Robert", OTP: " "})
	req := httptest.NewRequest(http.MethodPost, "/otp/validate", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	handler.ValidateOTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestValidateOTPMapsDomainError(t *testing.T) {
	handler := NewHandler(stubService{
		validateFunc: func(context.Context, string, string) (otpapp.ValidateResult, error) {
			return otpapp.ValidateResult{}, coreotp.ErrInvalidCode
		},
	}, zerolog.Nop())
	body, _ := json.Marshal(ValidateOTPRequest{UserID: "Robert", OTP: "12345"})
	req := httptest.NewRequest(http.MethodPost, "/otp/validate", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	handler.ValidateOTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errorBody := response["error"].(map[string]any)
	if errorBody["message"] != "otp invalid" {
		t.Fatalf("expected validation message from domain, got %v", errorBody["message"])
	}
}

func TestValidateOTPSuccess(t *testing.T) {
	var capturedUserID string
	var capturedOTP string
	handler := NewHandler(stubService{
		validateFunc: func(_ context.Context, userID, otp string) (otpapp.ValidateResult, error) {
			capturedUserID = userID
			capturedOTP = otp
			return otpapp.ValidateResult{UserID: userID}, nil
		},
	}, zerolog.Nop())
	body, _ := json.Marshal(ValidateOTPRequest{UserID: "Robert", OTP: "12345"})
	req := httptest.NewRequest(http.MethodPost, "/otp/validate", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	handler.ValidateOTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if capturedUserID != "Robert" || capturedOTP != "12345" {
		t.Fatalf("unexpected service input: %s %s", capturedUserID, capturedOTP)
	}

	var response ValidateOTPResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Data.Message != "OTP validated successfully." {
		t.Fatalf("unexpected success message: %s", response.Data.Message)
	}
}
