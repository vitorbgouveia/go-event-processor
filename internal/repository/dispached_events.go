package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/vitorbgouveia/go-event-processor/pkg/contracts"
)

type dispatchedEvents struct {
	client *dynamodb.Client
}

type DispatchedEvents interface {
	Insert(ctx context.Context, event contracts.EventMessageBody) error
}

const (
	tableName = "dispatched_events"
)

func NewDispatchedEvents(ctx context.Context) (DispatchedEvents, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg)

	return &dispatchedEvents{
		client,
	}, nil
}

func (s *dispatchedEvents) Insert(ctx context.Context, event contracts.EventMessageBody) error {
	newItem, err := attributevalue.MarshalMap(event)
	if err != nil {
		return err
	}

	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      newItem,
		TableName: aws.String(tableName),
	}); err != nil {
		return err
	}

	return nil
}
