package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/charmingruby/erebor/validator"
)

// ErrorResponse describes an API error and optional field validation failures.
type ErrorResponse struct {
	Message string                 `json:"message"`
	Fields  []validator.FieldError `json:"fields,omitempty"`
}

// WriteOKResponse writes v as JSON with status 200.
func WriteOKResponse(w http.ResponseWriter, v any) {
	writeResponse(w, http.StatusOK, v)
}

// WriteCreatedResponse writes v as JSON with status 201.
func WriteCreatedResponse(w http.ResponseWriter, v any) {
	writeResponse(w, http.StatusCreated, v)
}

// WriteEmptyResponse writes status 204 without a response body.
func WriteEmptyResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteBadRequest writes the error message as JSON with status 400.
func WriteBadRequest(w http.ResponseWriter, _ *http.Request, err error) {
	writeResponse(w, http.StatusBadRequest, err.Error())
}

// WriteErrorResponse writes an error response as JSON with the supplied status.
func WriteErrorResponse(w http.ResponseWriter, status int, response ErrorResponse) {
	writeResponse(w, status, response)
}

func writeResponse(w http.ResponseWriter, status int, v any) {
	if v == nil {
		w.WriteHeader(status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	body, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"Internal Server Error"}`))
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(body)
}
