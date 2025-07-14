package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	controllers "eventify/internal/api/controllers/v1"
	"eventify/internal/auth"
	"eventify/internal/domain"
	service_mocks "eventify/internal/service/mocks"
	"eventify/pkg/logger"
	"net/http/httptest"
	"testing"
	"time"

	"eventify/internal/constants"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Add this helper function to generate valid JWT tokens
func generateValidToken(jwtProvider auth.IJWTProvider) (string, error) {
	claims := jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"iss": "test-issuer",
		"aud": "test-audience",
		"permissions": []string{
			constants.Permissions.EventPermissions.Read,
			constants.Permissions.EventPermissions.Create,
			constants.Permissions.EventPermissions.Update,
			constants.Permissions.EventPermissions.Delete,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("test-secret-key"))
}

func setupTest(t *testing.T) (*controllers.EventController, *service_mocks.MockIEventService, *fiber.App, auth.IJWTProvider) {
	ctrl := gomock.NewController(t)
	mockService := service_mocks.NewMockIEventService(ctrl)
	app := fiber.New()
	log := logger.New(true)

	// Create a JWT provider for testing
	jwtProvider := auth.NewJWTProvider("test-secret-key", 1, "test-issuer", "test-audience")

	eventController := controllers.NewEventController(app, mockService, jwtProvider, log)
	eventController.RegisterRoutes()

	return eventController, mockService, app, jwtProvider
}

func TestGetAllEvents(t *testing.T) {
	_, mockService, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := generateValidToken(jwtProvider)
	require.NoError(t, err)

	// Create test data
	events := []domain.Event{
		{
			Id:        uuid.New(),
			Name:      "Test Event 1",
			Location:  "Location 1",
			Date:      time.Now(),
			CreatedBy: uuid.New(),
		},
		{
			Id:        uuid.New(),
			Name:      "Test Event 2",
			Location:  "Location 2",
			Date:      time.Now(),
			CreatedBy: uuid.New(),
		},
	}

	// Set up expectations
	mockService.EXPECT().
		GetAllEvents(context.Background()).
		Return(events)

	// Create test request
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	req.Header.Set("Authorization", "Bearer "+token) // Use the generated token

	// Perform the request
	resp, err := app.Test(req)

	// Assertions
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Parse response body
	var responseEvents []domain.Event
	err = json.NewDecoder(resp.Body).Decode(&responseEvents)
	require.NoError(t, err)
	require.Equal(t, len(events), len(responseEvents))
}

func TestGetEventById(t *testing.T) {
	_, mockService, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := generateValidToken(jwtProvider)
	require.NoError(t, err)

	eventId := uuid.New()
	event := &domain.Event{
		Id:        eventId,
		Name:      "Test Event",
		Location:  "Test Location",
		Date:      time.Now(),
		CreatedBy: uuid.New(),
	}

	// Set up expectations
	mockService.EXPECT().
		GetEventById(eventId, context.Background()).
		Return(event)

	// Create test request
	req := httptest.NewRequest("GET", "/api/v1/events/"+eventId.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token) // Use the generated token

	// Perform the request
	resp, err := app.Test(req)

	// Assertions
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var responseEvent domain.Event
	err = json.NewDecoder(resp.Body).Decode(&responseEvent)
	require.NoError(t, err)
	require.Equal(t, event.Id, responseEvent.Id)
}

func TestCreateEvent(t *testing.T) {
	_, mockService, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := generateValidToken(jwtProvider)
	require.NoError(t, err)

	newEvent := &domain.Event{
		Name:      "New Test Event",
		Location:  "New Location",
		Date:      time.Now(),
		CreatedBy: uuid.New(),
	}

	// Set up expectations
	mockService.EXPECT().
		CreateEvent(gomock.Any(), gomock.Any()).
		Return(nil)

	// Create request body
	body, err := json.Marshal(newEvent)
	require.NoError(t, err)

	// Create test request
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request
	resp, err := app.Test(req)

	// Assertions
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestUpdateEvent(t *testing.T) {
	_, mockService, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := generateValidToken(jwtProvider)
	require.NoError(t, err)

	eventId := uuid.New()
	updateEvent := &domain.Event{
		Id:        eventId,
		Name:      "Updated Event",
		Location:  "Updated Location",
		Date:      time.Now(),
		CreatedBy: uuid.New(),
	}

	// Set up expectations
	mockService.EXPECT().
		UpdateEvent(gomock.Any(), context.Background()).
		Return(nil)

	// Create request body
	body, err := json.Marshal(updateEvent)
	require.NoError(t, err)

	// Create test request
	req := httptest.NewRequest("PUT", "/api/v1/events/"+eventId.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request
	resp, err := app.Test(req)

	// Assertions
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestDeleteEvent(t *testing.T) {
	_, mockService, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := generateValidToken(jwtProvider)
	require.NoError(t, err)

	eventId := uuid.New()

	// Set up expectations
	mockService.EXPECT().
		DeleteEvent(eventId, context.Background()).
		Return(nil)

	// Create test request
	req := httptest.NewRequest("DELETE", "/api/v1/events/"+eventId.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request
	resp, err := app.Test(req)

	// Assertions
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

// Add error case tests
func TestGetEventByIdNotFound(t *testing.T) {
	_, mockService, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := generateValidToken(jwtProvider)
	require.NoError(t, err)

	eventId := uuid.New()

	// Set up expectations for not found case
	mockService.EXPECT().
		GetEventById(eventId, context.Background()).
		Return(nil)

	// Create test request
	req := httptest.NewRequest("GET", "/api/v1/events/"+eventId.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request
	resp, err := app.Test(req)

	// Assertions
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestCreateEventInvalidInput(t *testing.T) {
	_, _, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := generateValidToken(jwtProvider)
	require.NoError(t, err)

	// Create invalid request body
	invalidBody := []byte(`{"name": 123}`) // Invalid type for name field

	// Create test request
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(invalidBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request
	resp, err := app.Test(req)

	// Assertions
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
