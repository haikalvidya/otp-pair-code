package healthhttp

import (
	"net/http"

	httpadapter "otp-pair-code-interview/internal/adapters/http"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// Healthz godoc
// @Summary Health check
// @Description Check whether the service process is running.
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	httpadapter.WriteSuccess(w, r, http.StatusOK, HealthData{Status: "ok"})
}
