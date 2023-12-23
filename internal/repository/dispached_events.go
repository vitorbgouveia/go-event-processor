package repository

import (
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

type (
	dispatchedEvents struct {
		persistence aws.Persistence
	}
	DispatchedEvents interface {
		Insert(event EventInsertInput) error
	}

	EventInsertInput struct {
		EventId string `json:"event_id"`
		Context string `json:"context"`
		Type    string `json:"type"`
		Tenant  string `json:"tenant"`
		Data    string `json:"data"`
	}
)

const (
	tableName = "dispatched_events"
)

func NewDispatchedEvents(persistence aws.Persistence) DispatchedEvents {
	return &dispatchedEvents{persistence}
}

func (s *dispatchedEvents) Insert(event EventInsertInput) error {
	return s.persistence.Insert(aws.InsertInput{
		TableName:   tableName,
		StructValue: event,
	})
}
