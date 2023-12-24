package repository

import (
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

type (
	events struct {
		persistence aws.Persistence
	}
	Events interface {
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

func NewDispatchedEvents(persistence aws.Persistence) Events {
	return &events{persistence}
}

func (s *events) Insert(event EventInsertInput) error {
	return s.persistence.Insert(aws.InsertInput{
		TableName:   tableName,
		StructValue: event,
	})
}
