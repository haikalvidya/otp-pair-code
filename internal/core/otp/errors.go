package otp

import "errors"

var (
	ErrInvalidUserID    = errors.New("invalid user_id")
	ErrInvalidOTPInput  = errors.New("invalid otp input")
	ErrInvalidCode      = errors.New("otp invalid")
	ErrAlreadyActive    = errors.New("otp already active")
	ErrNotFound         = errors.New("otp not found")
	ErrExpired          = errors.New("otp expired")
	ErrBlocked          = errors.New("otp blocked")
	ErrGenerationFailed = errors.New("otp generation failed")
)
