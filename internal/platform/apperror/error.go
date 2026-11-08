package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type Kind string

const (
	KindValidation      Kind = "validation"
	KindBadRequest      Kind = "bad_request"
	KindNotFound        Kind = "not_found"
	KindConflict        Kind = "conflict"
	KindUnauthorized    Kind = "unauthorized"
	KindForbidden       Kind = "forbidden"
	KindLocked          Kind = "locked"
	KindRateLimited     Kind = "rate_limited"
	KindMFARequired     Kind = "mfa_required"
	KindNotConfigured   Kind = "not_configured"
	KindUnavailable     Kind = "unavailable"
	KindPayloadTooLarge Kind = "payload_too_large"
	KindInternal        Kind = "internal"
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Field   string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Wrap(kind Kind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Cause: cause}
}

func Validation(field, message string) *Error {
	return &Error{Kind: KindValidation, Code: "validation_failed", Message: message, Field: field}
}

func NotFound(resource string) *Error {
	return New(KindNotFound, "not_found", resource+" was not found.")
}

func Conflict(message string) *Error {
	return New(KindConflict, "conflict", message)
}

func Unauthorized(message string) *Error {
	return New(KindUnauthorized, "invalid_credentials", message)
}

func Forbidden(message string) *Error {
	return New(KindForbidden, "permission_denied", message)
}

func NotConfigured(provider string) *Error {
	return New(KindNotConfigured, "provider_not_configured", provider+" is not configured.")
}

func HTTPStatus(err error) int {
	var appErr *Error
	if !errors.As(err, &appErr) {
		return http.StatusInternalServerError
	}
	switch appErr.Kind {
	case KindBadRequest:
		return http.StatusBadRequest
	case KindValidation:
		return http.StatusUnprocessableEntity
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindLocked:
		return http.StatusLocked
	case KindRateLimited:
		return http.StatusTooManyRequests
	case KindMFARequired:
		return http.StatusPreconditionRequired
	case KindNotConfigured:
		return http.StatusNotImplemented
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindPayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case KindInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
