package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"eventify.analytics/domain"
	// "github.com/grafadruid/go-druid"
	// "github.com/grafadruid/go-druid/builder/query"
	"github.com/sirupsen/logrus"
)


type CreateEventService struct {
	// Add any dependencies or configurations needed for the service
	logger *logrus.Logger
}

func NewCreateEventService(logger *logrus.Logger) *CreateEventService {
	return &CreateEventService{
		logger: logger,
	}
}

func (s *CreateEventService) CreateEvent(entity domain.EventCreated) error {
	// Implement the logic to create an event
	// This could involve validating the input, saving to a database, etc.
	s.logger.WithFields(logrus.Fields{
		"event_data": entity,
	}).Info("Creating new event")

	if err := save(entity); err != nil {
        s.logger.Info("error sending event: %v", err)
    }

	return nil // Return an error if something goes wrong
}

// func (s *CreateEventService) GetEventById(id string) (*domain.EventCreated, error) {
// 	// Implement the logic to retrieve an event by ID
// 	s.logger.WithField("event_id", id).Info("Retrieving event by ID")
	

// 	// Example: Fetch from a database or external service
// 	// For now, return a dummy event
// 	return &domain.EventCreated{
// 		Id:          id,
// 		Name:        "Sample Event",
// 		Type:        "Sample Type",
// 		CountryCode: "US",
// 		OccurredAt:  "2023-10-01T00:00:00Z",
// 		CreatedBy:   "system",
// 	}, nil
// }

func save(entity domain.EventCreated) error {
		taskID := "index_events_cebopfkn_2025-07-19T04:33:13.402Z"
    	url := fmt.Sprintf("http://localhost:9999/druid/indexer/v1/firehose/%s", taskID)

    jsonData, err := json.Marshal(entity)
    if err != nil {
        return err
    }

    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed with status: %s", resp.Status)
    }

    return nil
}