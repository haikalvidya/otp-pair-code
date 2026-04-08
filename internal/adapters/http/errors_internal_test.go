package httpadapter

import (
	"net/http"
	"testing"

	domainerror "otp-pair-code/internal/core/domainerror"
)

func TestMapDomainErrorCoversAllKnownKinds(t *testing.T) {
	tests := []struct {
		name           string
		kind           domainerror.Kind
		errMsg         string
		expectedStatus int
		expectedMsg    string
	}{
		{name: "validation", kind: domainerror.KindValidation, errMsg: "invalid user_id", expectedStatus: http.StatusBadRequest, expectedMsg: "invalid user_id"},
		{name: "conflict", kind: domainerror.KindConflict, errMsg: "otp already active", expectedStatus: http.StatusConflict, expectedMsg: "Request conflicts with current state"},
		{name: "not found", kind: domainerror.KindNotFound, errMsg: "otp not found", expectedStatus: http.StatusNotFound, expectedMsg: "Resource not found"},
		{name: "gone", kind: domainerror.KindGone, errMsg: "otp expired", expectedStatus: http.StatusGone, expectedMsg: "Resource is no longer available"},
		{name: "blocked", kind: domainerror.KindBlocked, errMsg: "otp blocked", expectedStatus: http.StatusTooManyRequests, expectedMsg: "Too many failed attempts"},
		{name: "internal", kind: domainerror.KindInternal, errMsg: "otp generation failed", expectedStatus: http.StatusInternalServerError, expectedMsg: "Internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := MapDomainError(domainerror.New("test_code", tt.kind, tt.errMsg))
			if apiErr.StatusCode != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, apiErr.StatusCode)
			}
			if apiErr.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, apiErr.Message)
			}
		})
	}
}
