package domain

import "errors"

// The kinds of error the API maps to statuses (respond): a process's sentinel errors are made of one of them.
var (
	ErrInvalid      = errors.New("invalid")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
)

// kindError says its message and is one of the kinds above for errors.Is.
type kindError struct {
	message string
	kind    error
}

func (e *kindError) Error() string { return e.message }
func (e *kindError) Unwrap() error { return e.kind }

// Invalid is an error of the input: 400.
func Invalid(message string) error { return &kindError{message, ErrInvalid} }

// Unauthorized is a missing or wrong credential: 401.
func Unauthorized(message string) error { return &kindError{message, ErrUnauthorized} }

// Forbidden is a credential that cannot do this: 403.
func Forbidden(message string) error { return &kindError{message, ErrForbidden} }

// NotFound is a missing item: 404.
func NotFound(message string) error { return &kindError{message, ErrNotFound} }
