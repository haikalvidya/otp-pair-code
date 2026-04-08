package httpadapter

type Meta struct {
	RequestID string `json:"request_id,omitempty" example:"5f4dcc3b-5aa7"`
}

type ErrorBody struct {
	Code    string            `json:"code" example:"otp_invalid"`
	Message string            `json:"message" example:"OTP is invalid"`
	Details map[string]string `json:"details,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta,omitempty"`
}
