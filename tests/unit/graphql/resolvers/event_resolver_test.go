package resolvers_test

import (
	"context"
	"errors"
	"eventify/api/graphql/models"
	"eventify/api/graphql/resolvers"
	"eventify/internal/domain"
	service_mocks "eventify/internal/service/mocks"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var ErrEventNotFound = errors.New("event not found")

func setupTest(t *testing.T) (*resolvers.Resolver, *service_mocks.MockIEventService, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockService := service_mocks.NewMockIEventService(ctrl)
	log := logger.New(true)
	telemetryAdapter := telemetry.NewTelemetryAdapter()

	resolver := resolvers.NewResolver(mockService, log, telemetryAdapter)
	return resolver, mockService, ctrl
}

func TestCreateEvent(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()

	input := models.CreateEventInput{
		Name:        "Test Event",
		Description: "Test Description",
		Date:        "2024-01-15T10:00:00Z",
		Location:    "Test Location",
		Organizer:   "Test Organizer",
		Category:    "Test Category",
		Tags:        []string{"test", "event"},
		Capacity:    100,
	}

	// Set up expectations
	mockService.EXPECT().
		CreateEvent(gomock.Any(), ctx).
		DoAndReturn(func(event *domain.Event, ctx context.Context) error {
			event.Id = eventID
			event.CreatedBy = userID
			event.CreatedAt = time.Now()
			return nil
		})

	// Call the resolver
	result, err := resolver.Mutation().CreateEvent(ctx, input)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, eventID.String(), result.EventID)
	require.Equal(t, "Event created successfully", result.Message)
}

func TestCreateEvent_Error(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	input := models.CreateEventInput{
		Name:        "Test Event",
		Description: "Test Description",
		Date:        "2024-01-15T10:00:00Z",
		Location:    "Test Location",
		Organizer:   "Test Organizer",
		Category:    "Test Category",
		Tags:        []string{"test", "event"},
		Capacity:    100,
	}

	// Set up expectations for error
	mockService.EXPECT().
		CreateEvent(gomock.Any(), ctx).
		Return(ErrEventNotFound)

	// Call the resolver
	result, err := resolver.Mutation().CreateEvent(ctx, input)

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, ErrEventNotFound, err)
}

func TestCreateEvent_InvalidDate(t *testing.T) {
	resolver, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	input := models.CreateEventInput{
		Name:        "Test Event",
		Description: "Test Description",
		Date:        "invalid-date",
		Location:    "Test Location",
		Organizer:   "Test Organizer",
		Category:    "Test Category",
		Tags:        []string{"test", "event"},
		Capacity:    100,
	}

	// Call the resolver
	result, err := resolver.Mutation().CreateEvent(ctx, input)

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "parsing time")
}

func TestUpdateEvent(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()

	input := models.UpdateEventInput{
		Name:        stringPtr("Updated Event"),
		Description: stringPtr("Updated Description"),
		Date:        stringPtr("2024-01-15T10:00:00Z"),
		Location:    stringPtr("Updated Location"),
		Organizer:   stringPtr("Updated Organizer"),
		Category:    stringPtr("Updated Category"),
		Tags:        []string{"updated", "event"},
		Capacity:    intPtr(200),
	}

	existingEvent := &domain.Event{
		Id:          eventID,
		Name:        "Original Event",
		Description: "Original Description",
		Date:        time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
		Location:    "Original Location",
		Organizer:   "Original Organizer",
		Category:    "Original Category",
		Tags:        []string{"original", "event"},
		Capacity:    100,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}

	// Set up expectations
	mockService.EXPECT().
		GetEventById(eventID, ctx).
		Return(existingEvent)

	mockService.EXPECT().
		UpdateEvent(gomock.Any(), ctx).
		Return(nil)

	// Call the resolver
	result, err := resolver.Mutation().UpdateEvent(ctx, eventID.String(), input)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Updated Event", result.Event.Name)
	require.Equal(t, "Updated Description", result.Event.Description)
	require.Equal(t, "Updated Location", result.Event.Location)
	require.Equal(t, "Updated Organizer", result.Event.Organizer)
	require.Equal(t, "Updated Category", result.Event.Category)
	require.Equal(t, []string{"updated", "event"}, result.Event.Tags)
	require.Equal(t, 200, result.Event.Capacity)
	require.Equal(t, "Event updated successfully", result.Message)
}

func TestUpdateEvent_EventNotFound(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()

	input := models.UpdateEventInput{
		Name: stringPtr("Updated Event"),
	}

	// Set up expectations for event not found
	mockService.EXPECT().
		GetEventById(eventID, ctx).
		Return(nil)

	// Call the resolver
	result, err := resolver.Mutation().UpdateEvent(ctx, eventID.String(), input)

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "event not found")
}

func TestUpdateEvent_InvalidUUID(t *testing.T) {
	resolver, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	input := models.UpdateEventInput{
		Name: stringPtr("Updated Event"),
	}

	// Call the resolver with invalid UUID
	result, err := resolver.Mutation().UpdateEvent(ctx, "invalid-uuid", input)

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "invalid UUID")
}

func TestUpdateEvent_InvalidDate(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()

	input := models.UpdateEventInput{
		Date: stringPtr("invalid-date"),
	}

	existingEvent := &domain.Event{
		Id:        eventID,
		Name:      "Original Event",
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}

	// Set up expectations
	mockService.EXPECT().
		GetEventById(eventID, ctx).
		Return(existingEvent)

	// Call the resolver
	result, err := resolver.Mutation().UpdateEvent(ctx, eventID.String(), input)

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "parsing time")
}

func TestDeleteEvent(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()

	// Set up expectations
	mockService.EXPECT().
		DeleteEvent(eventID, ctx).
		Return(nil)

	// Call the resolver
	result, err := resolver.Mutation().DeleteEvent(ctx, eventID.String())

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "Event deleted successfully", result.Message)
}

func TestDeleteEvent_Error(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()

	// Set up expectations for error
	mockService.EXPECT().
		DeleteEvent(eventID, ctx).
		Return(ErrEventNotFound)

	// Call the resolver
	result, err := resolver.Mutation().DeleteEvent(ctx, eventID.String())

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, ErrEventNotFound, err)
}

func TestDeleteEvent_InvalidUUID(t *testing.T) {
	resolver, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Call the resolver with invalid UUID
	result, err := resolver.Mutation().DeleteEvent(ctx, "invalid-uuid")

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "invalid UUID")
}

func TestGetEvent(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()
	userID := uuid.New()

	expectedEvent := &domain.Event{
		Id:          eventID,
		Name:        "Test Event",
		Description: "Test Description",
		Date:        time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Location:    "Test Location",
		Organizer:   "Test Organizer",
		Category:    "Test Category",
		Tags:        []string{"test", "event"},
		Capacity:    100,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}

	// Set up expectations
	mockService.EXPECT().
		GetEventById(eventID, ctx).
		Return(expectedEvent)

	// Call the resolver
	result, err := resolver.Query().Event(ctx, eventID.String())

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, eventID, result.Id)
	require.Equal(t, "Test Event", result.Name)
	require.Equal(t, "Test Description", result.Description)
	require.Equal(t, "Test Location", result.Location)
	require.Equal(t, "Test Organizer", result.Organizer)
	require.Equal(t, "Test Category", result.Category)
	require.Equal(t, []string{"test", "event"}, result.Tags)
	require.Equal(t, 100, result.Capacity)
}

func TestGetEvent_NotFound(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	eventID := uuid.New()

	// Set up expectations for event not found
	mockService.EXPECT().
		GetEventById(eventID, ctx).
		Return(nil)

	// Call the resolver
	result, err := resolver.Query().Event(ctx, eventID.String())

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "event not found")
}

func TestGetEvent_InvalidUUID(t *testing.T) {
	resolver, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Call the resolver with invalid UUID
	result, err := resolver.Query().Event(ctx, "invalid-uuid")

	// Assertions
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "invalid UUID")
}

func TestGetEvents(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uuid.New()

	expectedEvents := []domain.Event{
		{
			Id:          uuid.New(),
			Name:        "Event 1",
			Description: "Description 1",
			Date:        time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Location:    "Location 1",
			Organizer:   "Organizer 1",
			Category:    "Category 1",
			Tags:        []string{"event1"},
			Capacity:    100,
			CreatedBy:   userID,
			CreatedAt:   time.Now(),
		},
		{
			Id:          uuid.New(),
			Name:        "Event 2",
			Description: "Description 2",
			Date:        time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC),
			Location:    "Location 2",
			Organizer:   "Organizer 2",
			Category:    "Category 2",
			Tags:        []string{"event2"},
			Capacity:    200,
			CreatedBy:   userID,
			CreatedAt:   time.Now(),
		},
	}

	// Set up expectations
	mockService.EXPECT().
		GetAllEvents(ctx).
		Return(expectedEvents)

	// Call the resolver
	result, err := resolver.Query().Events(ctx, intPtr(1), intPtr(10))

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Events, 2)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 1, result.Page)
	require.Equal(t, 10, result.Limit)
	require.Equal(t, "Event 1", result.Events[0].Name)
	require.Equal(t, "Event 2", result.Events[1].Name)
}

func TestGetEvents_EmptyList(t *testing.T) {
	resolver, mockService, ctrl := setupTest(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Set up expectations for empty list
	mockService.EXPECT().
		GetAllEvents(ctx).
		Return([]domain.Event{})

	// Call the resolver
	result, err := resolver.Query().Events(ctx, nil, nil)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Events, 0)
	require.Equal(t, 0, result.Total)
	require.Equal(t, 0, result.Page)
	require.Equal(t, 0, result.Limit)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
