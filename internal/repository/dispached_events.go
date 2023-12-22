package repository

import (
	"context"

	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

type (
	dispatchedEvents struct {
		persistence aws.Persistence
	}
	DispatchedEvents interface {
		Insert(ctx context.Context, event EventInsertInput) error
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

func NewDispatchedEvents(ctx context.Context, persistence aws.Persistence) DispatchedEvents {
	return &dispatchedEvents{persistence}
}

func (s *dispatchedEvents) Insert(ctx context.Context, event EventInsertInput) error {
	return s.persistence.Insert(ctx, aws.InsertInput{
		TableName:   tableName,
		StructValue: event,
	})
}
