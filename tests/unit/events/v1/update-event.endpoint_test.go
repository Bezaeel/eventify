package v1_test

import (
	"bytes"
	"encoding/json"
	controller "eventify/api/controllers/events/v1"
	"eventify/internal/auth"
	"eventify/internal/domain"
	service_mocks "eventify/internal/service/mocks"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"
	"eventify/tests/unit/helpers"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*controller.V1EventController, *service_mocks.MockIEventService, *fiber.App, auth.IJWTProvider) {
	ctrl := gomock.NewController(t)
	mockService := service_mocks.NewMockIEventService(ctrl)
	app := fiber.New()
	log := logger.New(true)
	telemetryAdapter := telemetry.NewTelemetryAdapter()

	// Create a JWT provider for testing
	jwtProvider := auth.NewJWTProvider("test-secret-key", 1, "test-issuer", "test-audience")

	v1EventController := controller.NewV1EventController(app, telemetryAdapter, mockService, jwtProvider, log)
	v1EventController.RegisterV1Routes()

	return v1EventController, mockService, app, jwtProvider
}

func TestUpdateEvent(t *testing.T) {
	_, mockService, app, jwtProvider := setupTest(t)

	// Generate valid token
	token, err := helpers.GenerateValidToken(jwtProvider, []string{"events.update"})
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
		UpdateEvent(gomock.Any(), gomock.Any()).
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
