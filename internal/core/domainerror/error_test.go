package domainerror

import (
	"errors"
	"testing"
)

func TestDetailOfReturnsJoinedDomainErrorDetails(t *testing.T) {
	err := errors.Join(New("otp_invalid", KindValidation, "otp invalid"), errors.New("driver failure"))

	detail, ok := DetailOf(err)
	if !ok {
		t.Fatal("expected detail to be found")
	}
	if detail.Code != "otp_invalid" || detail.Kind != KindValidation || detail.Message != "otp invalid" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
}

func TestDetailOfReturnsFalseForNonDomainError(t *testing.T) {
	_, ok := DetailOf(errors.New("plain error"))
	if ok {
		t.Fatal("expected false for non-domain error")
	}
}

func TestErrorReturnsMessage(t *testing.T) {
	err := New("otp_invalid", KindValidation, "otp invalid")
	if err.Error() != "otp invalid" {
		t.Fatalf("expected otp invalid, got %q", err.Error())
	}
}
