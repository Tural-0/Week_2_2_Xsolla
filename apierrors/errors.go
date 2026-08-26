package apierrors

import (
	"encoding/json"
	"net/http"
)

const (
	CodeInvalidRequest        = "INVALID_REQUEST"
	CodeValidationError       = "VALIDATION_ERROR"
	CodeBusinessRuleViolation = "BUSINESS_RULE_VIOLATION"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeForbidden             = "FORBIDDEN"
	CodeNotFound              = "NOT_FOUND"
	CodeConflict              = "CONFLICT"
	CodeInternal              = "INTERNAL_ERROR"
	CodeNotAllowed            = "NOT_ALLOWED"
)

type Response struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Write(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(Response{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
