package httpadapter

type RequestOTPRequest struct {
	UserID string `json:"user_id" example:"Robert"`
}

type RequestOTPResponse struct {
	UserID string `json:"user_id" example:"Robert"`
	OTP    string `json:"otp" example:"61531"`
}

type ValidateOTPRequest struct {
	UserID string `json:"user_id" example:"Robert"`
	OTP    string `json:"otp" example:"61531"`
}

type ValidateOTPResponse struct {
	UserID  string `json:"user_id" example:"Robert"`
	Message string `json:"message" example:"OTP validated successfully."`
}

type ErrorResponse struct {
	Error            string `json:"error" example:"otp_invalid"`
	ErrorDescription string `json:"error_description" example:"OTP is invalid"`
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}
