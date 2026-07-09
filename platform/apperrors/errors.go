// Package apperrors defines transport-agnostic error semantics.
//
// A feature handler returns one of these. Each transport maps them to its own
// representation: HTTP to a status code and JSON envelope, gRPC to a
// codes.Code, GraphQL to an entry in the errors array. Handlers never know
// which transport invoked them, so they must never return an HTTP status.
package apperrors

import "errors"

// Kind classifies a failure independently of any wire protocol.
type Kind int

const (
	// Internal is the zero value: an unclassified failure.
	Internal Kind = iota
	// NotFound means the addressed resource does not exist.
	NotFound
	// Invalid means the request was malformed or failed validation.
	Invalid
	// Conflict means the write collided with existing state.
	Conflict
	// Unauthorized means the caller is unauthenticated.
	Unauthorized
	// Forbidden means the caller is authenticated but lacks permission.
	Forbidden
)

// Error carries a Kind alongside a message and an optional wrapped cause.
type Error struct {
	cause   error
	Message string
	Kind    Kind
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// New builds an Error of the given Kind.
func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, Message: msg}
}

// Wrap builds an Error of the given Kind around an existing cause.
func Wrap(kind Kind, msg string, cause error) *Error {
	return &Error{Kind: kind, Message: msg, cause: cause}
}

// KindOf reports the Kind of err, defaulting to Internal for errors that did
// not originate here. A nil error is Internal; callers check err != nil first.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return Internal
}
