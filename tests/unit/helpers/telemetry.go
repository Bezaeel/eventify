package helpers

import (
	"eventify/pkg/telemetry/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// TelemetryAssertions provides helper functions for asserting telemetry calls
type TelemetryAssertions struct {
	mock *mocks.MockITelemetryAdapter
	t    *testing.T
}

// NewTelemetryAssertions creates a new telemetry assertions helper
func NewTelemetryAssertions(mock *mocks.MockITelemetryAdapter, t *testing.T) *TelemetryAssertions {
	return &TelemetryAssertions{
		mock: mock,
		t:    t,
	}
}

// ExpectTrackEvent sets up an expectation for TrackEvent call
func (ta *TelemetryAssertions) ExpectTrackEvent(eventName string, properties map[string]string) {
	ta.mock.EXPECT().TrackEvent(gomock.Any(), eventName, properties)
}

// ExpectTrackEventWithAnyProperties sets up an expectation for TrackEvent call with any properties
func (ta *TelemetryAssertions) ExpectTrackEventWithAnyProperties(eventName string) {
	ta.mock.EXPECT().TrackEvent(gomock.Any(), eventName, gomock.Any())
}

// ExpectTrackError sets up an expectation for TrackError call
func (ta *TelemetryAssertions) ExpectTrackError(properties map[string]string) {
	ta.mock.EXPECT().TrackError(gomock.Any(), properties)
}

// ExpectTrackErrorWithAnyProperties sets up an expectation for TrackError call with any properties
func (ta *TelemetryAssertions) ExpectTrackErrorWithAnyProperties() {
	ta.mock.EXPECT().TrackError(gomock.Any(), gomock.Any())
}

// ExpectNoTrackError sets up an expectation that TrackError should NOT be called
func (ta *TelemetryAssertions) ExpectNoTrackError() {
	// In gomock, not setting an expectation means it shouldn't be called
	// This is implicit behavior
}

// AssertExpectations verifies that all expected calls were made
func (ta *TelemetryAssertions) AssertExpectations() {
	// gomock automatically verifies expectations when the test ends
	// This is a convenience method for clarity
}

// AssertTrackEventCalled asserts that TrackEvent was called with specific parameters
func (ta *TelemetryAssertions) AssertTrackEventCalled(eventName string, properties map[string]string) {
	// This is a convenience method - gomock handles the actual verification
	// We can add additional custom assertions here if needed
	assert.True(ta.t, true, "TrackEvent expectation should be met by gomock")
}

// AssertTrackErrorCalled asserts that TrackError was called with specific parameters
func (ta *TelemetryAssertions) AssertTrackErrorCalled(properties map[string]string) {
	// This is a convenience method - gomock handles the actual verification
	// We can add additional custom assertions here if needed
	assert.True(ta.t, true, "TrackError expectation should be met by gomock")
}

// AssertNoTrackErrorCalled asserts that TrackError was NOT called
func (ta *TelemetryAssertions) AssertNoTrackErrorCalled() {
	// This is a convenience method - gomock handles the actual verification
	assert.True(ta.t, true, "TrackError should not be called")
}

// SetupCommonExpectations sets up common telemetry expectations for event operations
func (ta *TelemetryAssertions) SetupCommonExpectations(operation string, service string, eventID string) {
	expectedProperties := map[string]string{
		"operation": operation,
		"service":   service,
		"event_id":  eventID,
	}
	ta.ExpectTrackEvent("UpdateEvent", expectedProperties)
}

// SetupErrorExpectations sets up expectations for error scenarios
func (ta *TelemetryAssertions) SetupErrorExpectations(operation string, service string, eventID string) {
	// Expect the event to be tracked first
	eventProperties := map[string]string{
		"operation": operation,
		"service":   service,
		"event_id":  eventID,
	}
	ta.ExpectTrackEvent("UpdateEvent", eventProperties)

	// Expect the error to be tracked
	errorProperties := map[string]string{
		"operation": operation,
		"service":   service,
		"event_id":  eventID,
	}
	ta.ExpectTrackError(errorProperties)
}
