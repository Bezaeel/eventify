package service_test

import (
	"context"
	"eventify/internal/domain"
	"eventify/internal/service"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry/mocks"
	"eventify/tests/unit/helpers"
	"eventify/tests/unit/helpers/builders"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type EventServiceIntegrationTestSuite struct {
	suite.Suite
	db               *gorm.DB
	logger           *logger.Logger
	telemetryAdapter *mocks.MockITelemetryAdapter
	telemetryHelper  *helpers.TelemetryAssertions
	cleanUp          func()
	service          *service.EventService
	testUser         *domain.User
	testEvent        *domain.Event
}

func TestEventServiceSuite(t *testing.T) {
	suite.Run(t, new(EventServiceIntegrationTestSuite))
}

func (s *EventServiceIntegrationTestSuite) SetupSuite() {
	// Use the existing GetTestDB() function from setup_test.go
	db, cleanUp := baseIntegrationTest(s.T())
	s.db = db
	s.cleanUp = cleanUp
	s.logger = logger.New(true)
	ctrl := gomock.NewController(s.T())
	s.telemetryAdapter = mocks.NewMockITelemetryAdapter(ctrl)
	s.telemetryHelper = helpers.NewTelemetryAssertions(s.telemetryAdapter, s.T())
	s.service = service.NewEventService(s.db, s.logger, s.telemetryAdapter)
}

func (s *EventServiceIntegrationTestSuite) TearDownSuite() {
	s.cleanUp()
}

func (s *EventServiceIntegrationTestSuite) SetupTest() {
	// Create test user and event before each test
	s.testUser = &domain.User{
		ID:        uuid.New(),
		Email:     "test@mail.com",
		Password:  "password",
		FirstName: "Test",
		LastName:  "User",
		CreatedAt: time.Now(),
		UpdatedAt: nil,
	}

	// Reset the mock for each test
	ctrl := gomock.NewController(s.T())
	s.telemetryAdapter = mocks.NewMockITelemetryAdapter(ctrl)
	s.telemetryHelper = helpers.NewTelemetryAssertions(s.telemetryAdapter, s.T())
	s.service = service.NewEventService(s.db, s.logger, s.telemetryAdapter)
}

func (s *EventServiceIntegrationTestSuite) TearDownTest() {
	// Clean up the database after each test
	s.db.Exec("DELETE FROM events")
	s.db.Exec("DELETE FROM users")
}

func (s *EventServiceIntegrationTestSuite) TestCreateEvent() {
	// Ensure the test user exists in the database first
	err := s.db.Create(s.testUser).Error
	s.NoError(err)

	newEvent := builders.NewEventBuilder().
		WithDate(time.Now().Add(24 * time.Hour)).
		WithCreatedBy(s.testUser.ID).
		Create()
	err = s.service.CreateEvent(newEvent, context.Background())
	s.NoError(err)
}

func (s *EventServiceIntegrationTestSuite) TestGetEventById() {
	// First create the event
	testEvent := &domain.Event{
		Id:        uuid.New(),
		Name:      "Test Event",
		Date:      time.Now().Add(24 * time.Hour),
		Location:  "Test Location",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: nil,
		Creator:   *s.testUser,
	}
	err := s.db.Create(s.testUser) // Ensure the user is created first
	s.NoError(err.Error)
	err = s.db.Create(testEvent)
	s.NoError(err.Error)

	// Then test getting it
	event := s.service.GetEventById(testEvent.Id, context.Background())
	s.NotNil(event)
	s.Equal(testEvent.Id, event.Id)
	s.Equal(testEvent.Name, event.Name)
	s.Equal(testEvent.Location, event.Location)
}

func (s *EventServiceIntegrationTestSuite) TestGetAllEvents() {
	// Set up expectations using the helper
	s.telemetryHelper.ExpectTrackEvent("GetAllEvents", map[string]string{
		"operation": "fetch_all_events",
		"service":   "EventService",
	})

	// First create the event
	testEvents := []*domain.Event{
		{
			Id:        uuid.New(),
			Name:      "Test Event",
			Date:      time.Now().Add(24 * time.Hour),
			Location:  "Test Location",
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: nil,
			Creator:   *s.testUser,
		},
		{
			Id:        uuid.New(),
			Name:      "Another Test Event",
			Date:      time.Now().Add(48 * time.Hour),
			Location:  "Another Test Location",
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: nil,
			Creator:   *s.testUser,
		},
	}

	s.db.Create(s.testUser) // Ensure the user is created first
	err := s.db.Create(testEvents)
	s.NoError(err.Error)

	events := s.service.GetAllEvents(context.Background())
	s.NotNil(events)
	s.Equal(len(testEvents), len(events))
	for _, testEvent := range testEvents {
		found := false
		for _, event := range events {
			if event.Id == testEvent.Id {
				found = true
				break
			}
		}
		s.True(found, "Event %s not found in the list of events", testEvent.Id)
	}

	// Assert that expectations were met
	s.telemetryHelper.AssertExpectations()
}

func (s *EventServiceIntegrationTestSuite) TestUpdateEvent() {
	// First create the event
	initialEvent := &domain.Event{
		Id:        uuid.New(),
		Name:      "Test Event",
		Date:      time.Now().Add(24 * time.Hour),
		Location:  "Test Location",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: nil,
		Creator:   *s.testUser,
	}

	s.db.Create(s.testUser) // Ensure the user is created first
	s.db.Create(initialEvent)

	now := time.Now().UTC()
	updatedEvent := &domain.Event{
		Id:        initialEvent.Id,
		Name:      "Updated Event Name",
		Date:      time.Now().Add(24 * time.Hour),
		Location:  "Test Location",
		CreatedBy: initialEvent.Creator.ID,
		CreatedAt: initialEvent.CreatedAt,
		UpdatedAt: &now,
		Creator:   initialEvent.Creator,
	}

	// Set up expectations using the helper
	s.telemetryHelper.SetupCommonExpectations("update_event", "EventService", updatedEvent.Id.String())

	err := s.service.UpdateEvent(updatedEvent, context.Background())
	s.NoError(err)

	// Verify update
	var actual domain.Event
	s.db.First(&actual, "id = ?", initialEvent.Id)

	s.NotNil(actual)
	s.Equal(updatedEvent.Name, actual.Name)
	s.Equal(updatedEvent.Location, actual.Location)

	// Assert that expectations were met
	s.telemetryHelper.AssertExpectations()
}

func (s *EventServiceIntegrationTestSuite) TestUpdateEventWithError() {
	// Create an event with an invalid ID that doesn't exist in the database
	nonExistentEvent := &domain.Event{
		Id:        uuid.New(), // This ID doesn't exist in the database
		Name:      "Non-existent Event",
		Date:      time.Now().Add(24 * time.Hour),
		Location:  "Test Location",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: nil,
	}

	// Set up expectations for both event and error tracking using the helper
	s.telemetryHelper.SetupErrorExpectations("update_event", "EventService", nonExistentEvent.Id.String())

	// Attempt to update the non-existent event
	err := s.service.UpdateEvent(nonExistentEvent, context.Background())
	s.Error(err) // Should fail because the event doesn't exist

	// Assert that expectations were met
	s.telemetryHelper.AssertExpectations()
}

func (s *EventServiceIntegrationTestSuite) TestDeleteEvent() {
	// First create the event
	event := &domain.Event{
		Id:        uuid.New(),
		Name:      "Test Event",
		Date:      time.Now().Add(24 * time.Hour),
		Location:  "Test Location",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: nil,
		Creator:   *s.testUser,
	}

	s.db.Create(s.testUser) // Ensure the user is created first
	s.db.Create(event)

	s.service.DeleteEvent(event.Id, context.Background())

	// Verify update
	var actual domain.Event
	result := s.db.First(&actual, "id = ?", event.Id)

	s.Error(result.Error)
	s.Equal(result.Error, gorm.ErrRecordNotFound)
}
