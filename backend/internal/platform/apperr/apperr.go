// Package apperr defines domain errors that the HTTP layer maps to status
// codes and the {"error":{"code","message"}} response body.
package apperr

import "errors"

type Kind int

const (
	KindInvalid Kind = iota + 1
	KindNotFound
	KindUnauthorized
	KindForbidden
	KindConflict
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func Invalid(code, msg string) error   { return &Error{KindInvalid, code, msg} }
func NotFound(code, msg string) error  { return &Error{KindNotFound, code, msg} }
func Forbidden(code, msg string) error { return &Error{KindForbidden, code, msg} }
func Conflict(code, msg string) error  { return &Error{KindConflict, code, msg} }
func Unauthorized(code, msg string) error {
	return &Error{KindUnauthorized, code, msg}
}

// As extracts an *Error from err's chain.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
