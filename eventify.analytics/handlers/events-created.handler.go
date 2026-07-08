package handlers

import (
	"encoding/json"

	"eventify.analytics/domain"
	"eventify.analytics/events"
	"eventify.analytics/services"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type EventCreatedHandler struct {
	logger *logrus.Logger
}

func NewEventCreatedHandler(logger *logrus.Logger) *EventCreatedHandler {
	return &EventCreatedHandler{
		logger: logger,
	}
}

func (h *EventCreatedHandler) Handle(msg *message.Message) error {
	var EventCreated events.EventCreated
	if err := json.Unmarshal(msg.Payload, &EventCreated); err != nil {
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"messageId":  EventCreated.MessageId.String(),
		"eventName":  EventCreated.Name,
		"created_by": EventCreated.CreatedBy,
	}).Info("Processing course created event")

	// map the event to the domain entity
	eventEntity := &domain.EventCreated{
		Id:          uuid.New(),
		MessageId:   EventCreated.MessageId,
		Name:        EventCreated.Name,
		Type:        EventCreated.Type,
		CountryCode: EventCreated.CountryCode,
		OccurredAt:  EventCreated.OccurredAt,
		CreatedBy:   EventCreated.CreatedBy,
	}

	services.NewCreateEventService(h.logger).CreateEvent(*eventEntity)

	return nil
}
