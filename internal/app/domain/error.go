// Package domain holds the entities, validation rules and errors of the
// cantcp daemon and client. It depends on the standard library only and knows
// nothing about sockets, TLS implementations or configuration sources.
package domain

import "fmt"

// TypeError classifies a domain error so callers can react to the kind of
// failure without matching message strings.
type TypeError uint8

const (
	// ErrorInvalid marks a validation error of user input or configuration.
	ErrorInvalid TypeError = iota + 1
	// ErrorApp marks an application-level failure.
	ErrorApp
	// ErrorEarly marks a dependency that is not ready yet.
	ErrorEarly
)

// Error is a domain error: every error that crosses the domain boundary is
// wrapped into it and carries a TypeError.
type Error interface {
	error
	Type() TypeError
}

// domainError is the only Error implementation.
type domainError struct {
	typ TypeError
	msg string
}

// Error returns the message.
func (e domainError) Error() string { return e.msg }

// Type returns the error type.
func (e domainError) Type() TypeError { return e.typ }

// ErrInvalid builds a validation error: the message names the offending field
// and, where useful, the accepted values.
func ErrInvalid(format string, args ...any) Error {
	return domainError{typ: ErrorInvalid, msg: fmt.Sprintf(format, args...)}
}

// ErrApp builds an application-level error.
func ErrApp(format string, args ...any) Error {
	return domainError{typ: ErrorApp, msg: fmt.Sprintf(format, args...)}
}

// ErrEarly builds a dependency-not-ready error.
func ErrEarly(format string, args ...any) Error {
	return domainError{typ: ErrorEarly, msg: fmt.Sprintf(format, args...)}
}
