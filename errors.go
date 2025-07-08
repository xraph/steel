package forge_router

import (
	"fmt"
	"net/http"
	"time"
)

// APIError represents a structured API error
type APIError interface {
	Error() string
	StatusCode() int
	ErrorCode() string
	Details() interface{}
	ToResponse() ErrorResponse
}

// HTTPError is the main error type for API responses
type HTTPError struct {
	Status    int         `json:"status"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Detail    interface{} `json:"detail,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
	Path      string      `json:"path,omitempty"`
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Status, e.Code, e.Message)
}

func (e *HTTPError) StatusCode() int {
	return e.Status
}

func (e *HTTPError) ErrorCode() string {
	return e.Code
}

func (e *HTTPError) Details() interface{} {
	return e.Detail
}

func (e *HTTPError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Status:    e.Status,
			Code:      e.Code,
			Message:   e.Message,
			Detail:    e.Detail,
			Timestamp: e.Timestamp,
			RequestID: e.RequestID,
			Path:      e.Path,
		},
	}
}

// ErrorResponse is the standard error response format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Status    int         `json:"status" description:"HTTP status code"`
	Code      string      `json:"code" description:"Error code for programmatic handling"`
	Message   string      `json:"message" description:"Human-readable error message"`
	Detail    interface{} `json:"detail,omitempty" description:"Additional error details"`
	Timestamp time.Time   `json:"timestamp" description:"Error timestamp"`
	RequestID string      `json:"request_id,omitempty" description:"Request tracking ID"`
	Path      string      `json:"path,omitempty" description:"Request path that caused the error"`
}

// ValidationError represents validation errors with field-specific details
type ValidationError struct {
	HTTPError
	Fields []FieldError `json:"fields"`
}

type FieldError struct {
	Field   string      `json:"field" description:"Field name that failed validation"`
	Message string      `json:"message" description:"Validation error message"`
	Value   interface{} `json:"value,omitempty" description:"Value that failed validation"`
	Code    string      `json:"code,omitempty" description:"Validation error code"`
}

func (e *ValidationError) ToResponse() ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Status:    e.Status,
			Code:      e.Code,
			Message:   e.Message,
			Detail:    e.Fields,
			Timestamp: e.Timestamp,
			RequestID: e.RequestID,
			Path:      e.Path,
		},
	}
}

// Business logic errors
type BusinessError struct {
	HTTPError
	BusinessCode string      `json:"business_code"`
	Context      interface{} `json:"context,omitempty"`
}

// Common HTTP error constructors
func BadRequest(message string, details ...interface{}) *HTTPError {
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	return &HTTPError{
		Status:    http.StatusBadRequest,
		Code:      "BAD_REQUEST",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

func Unauthorized(message string, details ...interface{}) *HTTPError {
	if message == "" {
		message = "Authentication required"
	}
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	return &HTTPError{
		Status:    http.StatusUnauthorized,
		Code:      "UNAUTHORIZED",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

func Forbidden(message string, details ...interface{}) *HTTPError {
	if message == "" {
		message = "Access forbidden"
	}
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	return &HTTPError{
		Status:    http.StatusForbidden,
		Code:      "FORBIDDEN",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

func NotFound(resource string, details ...interface{}) *HTTPError {
	message := fmt.Sprintf("%s not found", resource)
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	return &HTTPError{
		Status:    http.StatusNotFound,
		Code:      "NOT_FOUND",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

func Conflict(message string, details ...interface{}) *HTTPError {
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	return &HTTPError{
		Status:    http.StatusConflict,
		Code:      "CONFLICT",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

func UnprocessableEntity(message string, fields ...FieldError) *ValidationError {
	if message == "" {
		message = "Validation failed"
	}
	return &ValidationError{
		HTTPError: HTTPError{
			Status:    http.StatusUnprocessableEntity,
			Code:      "VALIDATION_FAILED",
			Message:   message,
			Timestamp: time.Now(),
		},
		Fields: fields,
	}
}

func InternalServerError(message string, details ...interface{}) *HTTPError {
	if message == "" {
		message = "Internal server error"
	}
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	return &HTTPError{
		Status:    http.StatusInternalServerError,
		Code:      "INTERNAL_ERROR",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

func TooManyRequests(message string, retryAfter ...int) *HTTPError {
	if message == "" {
		message = "Too many requests"
	}
	var detail interface{}
	if len(retryAfter) > 0 {
		detail = map[string]interface{}{
			"retry_after_seconds": retryAfter[0],
		}
	}
	return &HTTPError{
		Status:    http.StatusTooManyRequests,
		Code:      "RATE_LIMITED",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

func ServiceUnavailable(message string, details ...interface{}) *HTTPError {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	var detail interface{}
	if len(details) > 0 {
		detail = details[0]
	}
	return &HTTPError{
		Status:    http.StatusServiceUnavailable,
		Code:      "SERVICE_UNAVAILABLE",
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

// Custom error for business logic
func NewBusinessError(status int, businessCode, message string, context interface{}) *BusinessError {
	return &BusinessError{
		HTTPError: HTTPError{
			Status:    status,
			Code:      "BUSINESS_ERROR",
			Message:   message,
			Timestamp: time.Now(),
		},
		BusinessCode: businessCode,
		Context:      context,
	}
}

// Helper for field validation errors
func NewFieldError(field, message string, value interface{}, codes ...string) FieldError {
	var code string
	if len(codes) > 0 {
		code = codes[0]
	}
	return FieldError{
		Field:   field,
		Message: message,
		Value:   value,
		Code:    code,
	}
}

// Response wrapper for successful responses with custom status codes
type APIResponse struct {
	StatusCode int
	Data       interface{}
	Headers    map[string]string
}

func NewResponse(statusCode int, data interface{}) *APIResponse {
	return &APIResponse{
		StatusCode: statusCode,
		Data:       data,
		Headers:    make(map[string]string),
	}
}

func (r *APIResponse) WithHeader(key, value string) *APIResponse {
	r.Headers[key] = value
	return r
}

// Success response helpers
func OK(data interface{}) *APIResponse {
	return NewResponse(http.StatusOK, data)
}

func Created(data interface{}) *APIResponse {
	return NewResponse(http.StatusCreated, data)
}

func Accepted(data interface{}) *APIResponse {
	return NewResponse(http.StatusAccepted, data)
}

func NoContent() *APIResponse {
	return NewResponse(http.StatusNoContent, nil)
}

// Enhanced FastContext with error helpers
func (c *FastContext) BadRequest(message string, details ...interface{}) error {
	return BadRequest(message, details...)
}

func (c *FastContext) Unauthorized(message string, details ...interface{}) error {
	return Unauthorized(message, details...)
}

func (c *FastContext) Forbidden(message string, details ...interface{}) error {
	return Forbidden(message, details...)
}

func (c *FastContext) NotFound(resource string, details ...interface{}) error {
	return NotFound(resource, details...)
}

func (c *FastContext) Conflict(message string, details ...interface{}) error {
	return Conflict(message, details...)
}

func (c *FastContext) ValidationError(message string, fields ...FieldError) error {
	return UnprocessableEntity(message, fields...)
}

func (c *FastContext) InternalError(message string, details ...interface{}) error {
	return InternalServerError(message, details...)
}

// OK Success response helpers for context
func (c *FastContext) OK(data interface{}) (*APIResponse, error) {
	return OK(data), nil
}

func (c *FastContext) Created(data interface{}) (*APIResponse, error) {
	return Created(data), nil
}

func (c *FastContext) Accepted(data interface{}) (*APIResponse, error) {
	return Accepted(data), nil
}

func (c *FastContext) NoContent() (*APIResponse, error) {
	return NoContent(), nil
}
