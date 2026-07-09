// Package events_test unit-tests the HTTP v1 event adapter.
//
// The adapter's whole job is decode -> call handler -> encode. These tests
// inject a stub handler and assert exactly that: status mapping, parameter
// validation, and response shape. No database, no Docker.
package events_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"eventify/api/internal/domain"
	"eventify/api/internal/features/events"
	v1events "eventify/api/internal/transport/http/v1/events"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// mount registers the controller on a bare app, bypassing JWT: authentication
// is the middleware's job and is tested separately.
func mount(t *testing.T, h v1events.Handlers) *fiber.App {
	t.Helper()
	c := v1events.New(h)
	app := fiber.New()
	app.Get("/events", c.List)
	app.Get("/events/:id", c.Get)
	app.Put("/events/:id", c.Update)
	app.Delete("/events/:id", c.Delete)
	return app
}

func do(t *testing.T, app *fiber.App, method, target string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, target, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	return resp
}

func validUpdateBody() map[string]any {
	return map[string]any{
		"name": "n", "description": "d", "location": "l",
		"date": time.Now().Format(time.RFC3339), "organizer": "o",
		"category": "c", "tags": []string{"t"}, "capacity": 1,
	}
}

// Every Kind a handler can return must map to a distinct status. This is the
// contract httperr.Status exists to hold.
func TestUpdate_MapsErrorKindToStatus(t *testing.T) {
	tests := []struct {
		name string
		kind apperrors.Kind
		want int
	}{
		{"not found", apperrors.NotFound, http.StatusNotFound},
		{"invalid", apperrors.Invalid, http.StatusBadRequest},
		{"conflict", apperrors.Conflict, http.StatusConflict},
		{"unauthorized", apperrors.Unauthorized, http.StatusUnauthorized},
		{"forbidden", apperrors.Forbidden, http.StatusForbidden},
		{"internal", apperrors.Internal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mount(t, v1events.Handlers{
				Update: func(context.Context, events.UpdateEventCommand) (events.UpdateEventResult, error) {
					return events.UpdateEventResult{}, apperrors.New(tt.kind, "boom")
				},
			})

			resp := do(t, app, http.MethodPut, "/events/"+uuid.NewString(), validUpdateBody())
			require.Equal(t, tt.want, resp.StatusCode)
		})
	}
}

func TestUpdate_InternalErrorDoesNotLeakDriverMessage(t *testing.T) {
	app := mount(t, v1events.Handlers{
		Update: func(context.Context, events.UpdateEventCommand) (events.UpdateEventResult, error) {
			return events.UpdateEventResult{}, apperrors.Wrap(apperrors.Internal, "update event",
				errRawDriver{})
		},
	})

	resp := do(t, app, http.MethodPut, "/events/"+uuid.NewString(), validUpdateBody())
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotContains(t, string(body), "pq: column \"secret_column\"",
		"internal errors must not expose schema details to clients")
}

type errRawDriver struct{}

func (errRawDriver) Error() string { return `pq: column "secret_column" does not exist` }

func TestUpdate_RejectsMalformedID(t *testing.T) {
	called := false
	app := mount(t, v1events.Handlers{
		Update: func(context.Context, events.UpdateEventCommand) (events.UpdateEventResult, error) {
			called = true
			return events.UpdateEventResult{}, nil
		},
	})

	resp := do(t, app, http.MethodPut, "/events/not-a-uuid", validUpdateBody())
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.False(t, called, "the handler must not run on a malformed id")
}

func TestUpdate_PassesEveryFieldToTheCommand(t *testing.T) {
	// The old v1 mapper silently dropped Description, Organizer, Category, Tags
	// and Capacity on the way to the write.
	var got events.UpdateEventCommand
	id := uuid.New()

	app := mount(t, v1events.Handlers{
		Update: func(_ context.Context, cmd events.UpdateEventCommand) (events.UpdateEventResult, error) {
			got = cmd
			return events.UpdateEventResult{EventID: cmd.EventID, UpdatedAt: time.Now()}, nil
		},
	})

	resp := do(t, app, http.MethodPut, "/events/"+id.String(), validUpdateBody())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Equal(t, id, got.EventID, "the path id must reach the command")
	require.Equal(t, "d", got.Description)
	require.Equal(t, "o", got.Organizer)
	require.Equal(t, "c", got.Category)
	require.Equal(t, []string{"t"}, got.Tags)
	require.Equal(t, 1, got.Capacity)
}

func TestList_ForwardsPagingAndEncodesTotal(t *testing.T) {
	var got events.GetEventsQuery
	app := mount(t, v1events.Handlers{
		List: func(_ context.Context, q events.GetEventsQuery) (events.GetEventsResult, error) {
			got = q
			return events.GetEventsResult{Events: []domain.Event{{ID: uuid.New()}}, Total: 37}, nil
		},
	})

	resp := do(t, app, http.MethodGet, "/events?limit=5&offset=10", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 5, got.Limit)
	require.Equal(t, 10, got.Offset)

	var body struct {
		Events []map[string]any `json:"events"`
		Total  int              `json:"total"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, 37, body.Total, "total is the row count, not the page size")
	require.Len(t, body.Events, 1)
}

func TestGet_UsesOrganizerSpelling(t *testing.T) {
	app := mount(t, v1events.Handlers{
		Get: func(context.Context, events.GetEventQuery) (domain.Event, error) {
			return domain.Event{ID: uuid.New(), Organizer: "Bezaeel"}, nil
		},
	})

	resp := do(t, app, http.MethodGet, "/events/"+uuid.NewString(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "Bezaeel", body["organizer"], "v1 spells it organizer")
	require.NotContains(t, body, "organiser", "the v2 spelling must not leak into v1")
}

func TestDelete_Returns204(t *testing.T) {
	app := mount(t, v1events.Handlers{
		Delete: func(context.Context, events.DeleteEventCommand) error { return nil },
	})

	resp := do(t, app, http.MethodDelete, "/events/"+uuid.NewString(), nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}
