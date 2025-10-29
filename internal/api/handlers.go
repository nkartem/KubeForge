package api

import (
	"encoding/json"
	"net/http"
)

// Response is a standard API response envelope
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes a successful JSON response
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// WriteCreated writes a 201 Created response
func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// WriteError writes an error JSON response
func WriteError(w http.ResponseWriter, statusCode int, code, message string) {
	WriteJSON(w, statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// WriteBadRequest writes a 400 Bad Request error
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// WriteNotFound writes a 404 Not Found error
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, "NOT_FOUND", message)
}

// WriteInternalError writes a 500 Internal Server Error
func WriteInternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", message)
}

// ParseJSON parses JSON request body into the given struct
func ParseJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
