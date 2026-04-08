package httpadapter

import (
	"net/http"

	domainerror "otp-pair-code/internal/core/domainerror"
)

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]string
}

func MapDomainError(err error) APIError {
	detail, ok := domainerror.DetailOf(err)
	if !ok {
		return APIError{StatusCode: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error"}
	}

	message := publicMessageFromErrorKind(detail.Kind)
	if detail.Kind == domainerror.KindValidation && detail.Message != "" {
		message = detail.Message
	}

	return APIError{
		StatusCode: httpStatusFromErrorKind(detail.Kind),
		Code:       detail.Code,
		Message:    message,
	}
}

func httpStatusFromErrorKind(kind domainerror.Kind) int {
	switch kind {
	case domainerror.KindValidation:
		return http.StatusBadRequest
	case domainerror.KindConflict:
		return http.StatusConflict
	case domainerror.KindNotFound:
		return http.StatusNotFound
	case domainerror.KindGone:
		return http.StatusGone
	case domainerror.KindBlocked:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func publicMessageFromErrorKind(kind domainerror.Kind) string {
	switch kind {
	case domainerror.KindValidation:
		return "Request is invalid"
	case domainerror.KindConflict:
		return "Request conflicts with current state"
	case domainerror.KindNotFound:
		return "Resource not found"
	case domainerror.KindGone:
		return "Resource is no longer available"
	case domainerror.KindBlocked:
		return "Too many failed attempts"
	default:
		return "Internal server error"
	}
}
