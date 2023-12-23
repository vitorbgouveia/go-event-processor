package aws

import (
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/vitorbgouveia/go-event-processor/pkg/logger"
)

type (
	persistence struct {
		client dynamodbiface.DynamoDBAPI
	}

	Persistence interface {
		Insert(i InsertInput) error
	}

	InsertInput struct {
		TableName   string
		StructValue interface{}
	}
)

var (
	ErrInvalidInsertInput = errors.New("input structValue is invalid, could not parse to map")
	ErrInvalidTableName   = errors.New("input table name is invalid")
)

func NewPersistence(sess *session.Session) Persistence {
	client := dynamodb.New(sess)
	return &persistence{client}
}

func (s *persistence) Insert(i InsertInput) error {
	newItem, err := dynamodbattribute.MarshalMap(i.StructValue)
	if err != nil || len(newItem) == 0 {
		return errors.Join(err, ErrInvalidInsertInput)
	}

	if i.TableName == "" {
		return ErrInvalidTableName
	}

	if _, err := s.client.PutItem(&dynamodb.PutItemInput{
		Item:      newItem,
		TableName: aws.String(i.TableName),
	}); err != nil {
		slog.Error("fail to insert new item", slog.String(logger.TableNameKey, i.TableName),
			slog.Any(logger.EventBodyKey, newItem), slog.Any("err", err))
		return err
	}

	return nil
}
