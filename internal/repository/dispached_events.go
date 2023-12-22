package repository

import (
	"context"

	"github.com/vitorbgouveia/go-event-processor/internal/models"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
)

type (
	dispatchedEvents struct {
		persistence aws.Persistence
	}
	DispatchedEvents interface {
		Insert(ctx context.Context, event models.EventMessageBody) error
	}
)

const (
	tableName = "dispatched_events"
)

func NewDispatchedEvents(ctx context.Context, persistence aws.Persistence) DispatchedEvents {
	return &dispatchedEvents{persistence}
}

func (s *dispatchedEvents) Insert(ctx context.Context, event models.EventMessageBody) error {
	return s.persistence.Insert(ctx, aws.InsertInput{
		TableName:   tableName,
		StructValue: event,
	})
}
