package aws

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/vitorbgouveia/go-event-processor/pkg/logger"
)

type (
	persistence struct {
		client *dynamodb.Client
	}

	Persistence interface {
		Insert(ctx context.Context, i InsertInput) error
	}

	InsertInput struct {
		TableName   string
		StructValue interface{}
	}
)

func NewPersistence(cfg aws.Config) Persistence {
	client := dynamodb.NewFromConfig(cfg)
	return &persistence{client}
}

func (s *persistence) Insert(ctx context.Context, i InsertInput) error {
	newItem, err := attributevalue.MarshalMap(i.StructValue)
	if err != nil {
		return err
	}

	if _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      newItem,
		TableName: aws.String(i.TableName),
	}); err != nil {
		slog.Error("fail to insert new item", slog.String(logger.TableNameKey, i.TableName),
			slog.Any(logger.EventBodyKey, i.StructValue), slog.Any("err", err))
		return err
	}

	return nil
}
