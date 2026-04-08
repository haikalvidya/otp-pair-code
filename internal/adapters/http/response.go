package httpadapter

import (
	"encoding/json"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func WriteSuccess(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, status, map[string]any{
		"data": data,
		"meta": Meta{RequestID: chimiddleware.GetReqID(r.Context())},
	})
}

func WriteAPIError(w http.ResponseWriter, r *http.Request, apiErr APIError) {
	writeJSON(w, apiErr.StatusCode, ErrorResponse{
		Error: ErrorBody{
			Code:    apiErr.Code,
			Message: apiErr.Message,
			Details: apiErr.Details,
		},
		Meta: Meta{RequestID: chimiddleware.GetReqID(r.Context())},
	})
}

func WriteDomainError(w http.ResponseWriter, r *http.Request, err error) {
	WriteAPIError(w, r, MapDomainError(err))
}

func DecodeJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
