package statusError

import (
	"fmt"
)

// StatusError is error status
type StatusError struct {
	// 外部显示的大类 error code,参与Is的比较
	code Code
	// 用于决定的是哪一种error,以及error的详细信息,参与Is的比较
	reason string
	// 外挂信息,比如剩余可重试次数、quota不足的数量，不参与Is的比较
	details []interface{}
}

// New error
func New(code Code, reason string, details ...interface{}) *StatusError {
	return &StatusError{
		code:    code,
		reason:  reason,
		details: details,
	}
}

// Error format error
func (se *StatusError) Error() string {
	return fmt.Sprintf("%d %s", se.Code(), se.Reason())
}

// Code is Status code
func (se *StatusError) Code() Code {
	if se == nil {
		return CodeOK
	}
	return se.code
}

// Is compare error
func (se *StatusError) Is(target error) bool {
	if se == nil || target == nil {
		if se == nil && target == nil {
			return true
		}
		return false
	}

	if err, ok := target.(*StatusError); ok {
		return (se.Code() == err.Code()) && (se.reason == err.reason)
	}
	return false
}

// Details is get Details
func (se *StatusError) Details() []interface{} {
	if se == nil {
		return nil
	}
	return se.details
}

// WithDetails append details
func (se *StatusError) WithDetails(details ...interface{}) {
	if se == nil {
		return
	}
	se.details = append(se.details, details...)
}

// Reason is get resaon
func (se *StatusError) Reason() string {
	if se == nil {
		return "OK"
	}
	return se.reason
}
