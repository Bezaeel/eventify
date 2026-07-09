// Package httperr maps transport-agnostic apperrors onto HTTP.
//
// ErrorResponse used to live in pkg/ErrorResponse.go, shared by "all three
// transports" — but a JSON envelope is meaningless to gRPC, which signals with
// status codes, and to GraphQL, which signals with an errors array. It belongs
// here, at the HTTP edge, and nowhere else.
package httperr

import (
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
)

// ErrorResponse is the JSON body returned for any failed HTTP request.
type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// Status maps a Kind to its HTTP status code. This is the only place in the
// codebase that knows a NotFound is a 404.
func Status(kind apperrors.Kind) int {
	switch kind {
	case apperrors.NotFound:
		return fiber.StatusNotFound
	case apperrors.Invalid:
		return fiber.StatusBadRequest
	case apperrors.Conflict:
		return fiber.StatusConflict
	case apperrors.Unauthorized:
		return fiber.StatusUnauthorized
	case apperrors.Forbidden:
		return fiber.StatusForbidden
	case apperrors.Internal:
		return fiber.StatusInternalServerError
	default:
		return fiber.StatusInternalServerError
	}
}

// Write renders err as JSON with the status its Kind implies.
//
// Internal errors deliberately do not leak err.Error() to the client: the old
// handlers returned raw driver messages in the body, which exposes table and
// column names. Callers should log the error; the client gets a generic string.
func Write(c *fiber.Ctx, err error) error {
	kind := apperrors.KindOf(err)
	status := Status(kind)

	body := ErrorResponse{Message: err.Error()}
	if kind == apperrors.Internal {
		body = ErrorResponse{Message: "internal server error"}
	}
	return c.Status(status).JSON(body)
}
