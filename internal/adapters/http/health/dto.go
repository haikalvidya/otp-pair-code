package healthhttp

import httpadapter "otp-pair-code/internal/adapters/http"

type HealthData struct {
	Status string `json:"status" example:"ok"`
}

type HealthResponse struct {
	Data HealthData       `json:"data"`
	Meta httpadapter.Meta `json:"meta,omitempty"`
}
