package httpx

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    any         `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Ok(w http.ResponseWriter, message string, data any) {
	JSON(w, http.StatusOK, APIResponse{Success: true, Message: message, Data: data})
}

func Created(w http.ResponseWriter, message string, data any) {
	JSON(w, http.StatusCreated, APIResponse{Success: true, Message: message, Data: data})
}

func Fail(w http.ResponseWriter, status int, code, details string) {
	JSON(w, status, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Details: details},
	})
}
