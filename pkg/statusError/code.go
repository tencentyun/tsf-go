package statusError

import "errors"

// Code is code
type Code int

var (
	CodeOK                   Code = 200
	CodeBadRequest           Code = 400
	CodeUnauthorized         Code = 401
	CodeForbidden            Code = 403
	CodeNotFound             Code = 404
	CodePreconditionRequired Code = 428
	CodeLimitExceed          Code = 429
	CodeServerClosed         Code = 444
	CodeClientClosed         Code = 499
	CodeInternal             Code = 500
	CodeNotImplemented       Code = 501
	CodeServiceUnavailable   Code = 503
	CodeDeadline             Code = 504
	CodeUnknown              Code = 520
)

// OK is 200
func OK(reason string, details ...interface{}) *StatusError {
	return New(CodeOK, reason, details...)
}

// IsOK return error is 200
func IsOK(err error) bool {
	if err == nil {
		return true
	}
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeOK
	}
	return false
}

// BadRequest is 400
func BadRequest(reason string, details ...interface{}) *StatusError {
	return New(CodeBadRequest, reason, details...)
}

// IsBadRequest return error is 400
func IsBadRequest(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeBadRequest
	}
	return false
}

// Unauthorized is 401
func Unauthorized(reason string, details ...interface{}) *StatusError {
	return New(CodeUnauthorized, reason, details...)
}

// IsUnauthorized return error is 401
func IsUnauthorized(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeUnauthorized
	}
	return false
}

// Forbidden is 403
func Forbidden(reason string, details ...interface{}) *StatusError {
	return New(CodeForbidden, reason, details...)
}

// IsForbidden return error is 403
func IsForbidden(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeForbidden
	}
	return false
}

// NotFound is 404
func NotFound(reason string, details ...interface{}) *StatusError {
	return New(CodeNotFound, reason, details...)
}

// IsNotFound return error is 404
func IsNotFound(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeNotFound
	}
	return false
}

// PreconditionRequired is 428
func PreconditionRequired(reason string, details ...interface{}) *StatusError {
	return New(CodePreconditionRequired, reason, details...)
}

// IsPreconditionRequired return error is 428
func IsPreconditionRequired(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodePreconditionRequired
	}
	return false
}

// LimitExceed is 429
func LimitExceed(reason string, details ...interface{}) *StatusError {
	return New(CodeLimitExceed, reason, details...)
}

// IsLimitExceed return error is 429
func IsLimitExceed(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeLimitExceed
	}
	return false
}

// ServerClosed is 444
func ServerClosed(reason string, details ...interface{}) *StatusError {
	return New(CodeServerClosed, reason, details...)
}

// IsServerClosed return error is 444
func IsServerClosed(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeServerClosed
	}
	return false
}

// ClientClosed is 499
func ClientClosed(reason string, details ...interface{}) *StatusError {
	return New(CodeClientClosed, reason, details...)
}

// IsClientClosed return error is 499
func IsClientClosed(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeClientClosed
	}
	return false
}

// Internal is 500
func Internal(reason string, details ...interface{}) *StatusError {
	return New(CodeInternal, reason, details...)
}

// IsInternal return error is 500
func IsInternal(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeInternal
	}
	return false
}

// NotImplemented is 501
func NotImplemented(reason string, details ...interface{}) *StatusError {
	return New(CodeNotImplemented, reason, details...)
}

// IsNotImplemented return error is 501
func IsNotImplemented(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeNotImplemented
	}
	return false
}

// ServiceUnavailable is 503
func ServiceUnavailable(reason string, details ...interface{}) *StatusError {
	return New(CodeServiceUnavailable, reason, details...)
}

// IsServiceUnavailable return error is 503
func IsServiceUnavailable(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeServiceUnavailable
	}
	return false
}

// Deadline is 504
func Deadline(reason string, details ...interface{}) *StatusError {
	return New(CodeDeadline, reason, details...)
}

// IsDeadline return error is 504
func IsDeadline(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeDeadline
	}
	return false
}

// Unknown is 520
func Unknown(reason string, details ...interface{}) *StatusError {
	return New(CodeUnknown, reason, details...)
}

// IsUnknown return error is 520
func IsUnknown(err error) bool {
	if se := new(StatusError); errors.As(err, &se) {
		return se.Code() == CodeUnknown
	}
	return false
}
