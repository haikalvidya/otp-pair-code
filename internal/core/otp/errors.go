package otp

import domainerror "otp-pair-code/internal/core/domainerror"

const (
	CodeInvalidRequest = "invalid_request"
	CodeAlreadyActive  = "otp_already_active"
	CodeNotFound       = "otp_not_found"
	CodeExpired        = "otp_expired"
	CodeInvalid        = "otp_invalid"
	CodeBlocked        = "otp_blocked"
	CodeInternal       = "internal_error"
)

var (
	ErrInvalidUserID    = domainerror.New(CodeInvalidRequest, domainerror.KindValidation, "invalid user_id")
	ErrInvalidOTPInput  = domainerror.New(CodeInvalidRequest, domainerror.KindValidation, "invalid otp input")
	ErrInvalidCode      = domainerror.New(CodeInvalid, domainerror.KindValidation, "otp invalid")
	ErrAlreadyActive    = domainerror.New(CodeAlreadyActive, domainerror.KindConflict, "otp already active")
	ErrNotFound         = domainerror.New(CodeNotFound, domainerror.KindNotFound, "otp not found")
	ErrExpired          = domainerror.New(CodeExpired, domainerror.KindGone, "otp expired")
	ErrBlocked          = domainerror.New(CodeBlocked, domainerror.KindBlocked, "otp blocked")
	ErrGenerationFailed = domainerror.New(CodeInternal, domainerror.KindInternal, "otp generation failed")
)
