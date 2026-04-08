package otphttp

import httpadapter "otp-pair-code/internal/adapters/http"

type RequestOTPRequest struct {
	UserID string `json:"user_id" example:"Robert"`
}

type RequestOTPData struct {
	UserID string `json:"user_id" example:"Robert"`
	OTP    string `json:"otp" example:"61531"`
}

type RequestOTPResponse struct {
	Data RequestOTPData   `json:"data"`
	Meta httpadapter.Meta `json:"meta,omitempty"`
}

type ValidateOTPRequest struct {
	UserID string `json:"user_id" example:"Robert"`
	OTP    string `json:"otp" example:"61531"`
}

type ValidateOTPData struct {
	UserID  string `json:"user_id" example:"Robert"`
	Message string `json:"message" example:"OTP validated successfully."`
}

type ValidateOTPResponse struct {
	Data ValidateOTPData  `json:"data"`
	Meta httpadapter.Meta `json:"meta,omitempty"`
}
