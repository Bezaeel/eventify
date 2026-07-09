// Package events_test unit-tests the HTTP v2 event adapter.
//
// v2 exists to prove the versioning rule: a wire-contract change is a new DTO
// and nothing more. These tests assert that v2's `organiser` spelling reaches
// the same UpdateEventCommand field that v1's `organizer` does — one handler,
// two contracts.
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
	v2events "eventify/api/internal/transport/http/v2/events"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mount(t *testing.T, h v2events.Handlers) *fiber.App {
	t.Helper()
	c := v2events.New(h)
	app := fiber.New()
	app.Get("/events", c.List)
	app.Put("/events/:id", c.Update)
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

func TestUpdate_OrganiserMapsToTheSameCommandField(t *testing.T) {
	var got events.UpdateEventCommand
	id := uuid.New()

	app := mount(t, v2events.Handlers{
		Update: func(_ context.Context, cmd events.UpdateEventCommand) (events.UpdateEventResult, error) {
			got = cmd
			return events.UpdateEventResult{EventID: cmd.EventID}, nil
		},
		Get: func(context.Context, events.GetEventQuery) (domain.Event, error) {
			return domain.Event{ID: id, Organizer: "Bezaeel"}, nil
		},
	})

	resp := do(t, app, http.MethodPut, "/events/"+id.String(), map[string]any{
		"name": "n", "description": "d", "location": "l",
		"date": time.Now().Format(time.RFC3339),
		// v2 spelling
		"organiser": "Bezaeel",
		"category":  "c", "tags": []string{"t"}, "capacity": 1,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// The command has no `Organiser`. The DTO difference dies at the adapter.
	require.Equal(t, "Bezaeel", got.Organizer)
}

func TestUpdate_ReturnsFullEventWithOrganiserSpelling(t *testing.T) {
	id := uuid.New()
	app := mount(t, v2events.Handlers{
		Update: func(_ context.Context, cmd events.UpdateEventCommand) (events.UpdateEventResult, error) {
			return events.UpdateEventResult{EventID: cmd.EventID}, nil
		},
		Get: func(context.Context, events.GetEventQuery) (domain.Event, error) {
			return domain.Event{ID: id, Organizer: "Bezaeel", Name: "n"}, nil
		},
	})

	resp := do(t, app, http.MethodPut, "/events/"+id.String(), map[string]any{
		"name": "n", "organiser": "Bezaeel", "capacity": 1,
		"date": time.Now().Format(time.RFC3339),
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "Bezaeel", body["organiser"], "v2 spells it organiser")
	require.NotContains(t, body, "organizer")
	// v1 returns only {event_id, updated_at}; v2 returns the whole event.
	require.Equal(t, "n", body["name"])
}

func TestUpdate_MapsNotFoundTo404(t *testing.T) {
	app := mount(t, v2events.Handlers{
		Update: func(context.Context, events.UpdateEventCommand) (events.UpdateEventResult, error) {
			return events.UpdateEventResult{}, apperrors.New(apperrors.NotFound, "event not found")
		},
	})

	resp := do(t, app, http.MethodPut, "/events/"+uuid.NewString(), map[string]any{"capacity": 1})
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
