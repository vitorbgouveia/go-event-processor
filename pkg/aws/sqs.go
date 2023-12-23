package aws

import (
	"log/slog"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/vitorbgouveia/go-event-processor/pkg/logger"
)

type (
	messagebroker struct {
		client sqsiface.SQSAPI
	}

	SendMessageInput struct {
		QueueName   string
		MessageID   string
		MessageBody string
	}

	MessageBroker interface {
		SendMessage(p SendMessageInput) error
	}
)

func NewMessageBroker(sess *session.Session) MessageBroker {
	client := sqs.New(sess)
	return &messagebroker{client}
}

func (s *messagebroker) SendMessage(p SendMessageInput) error {
	if err := p.Validate(); err != nil {
		return err
	}

	urlOutput, err := s.client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(p.QueueName),
	})
	if err != nil {
		slog.Error("could not get url queue", slog.String(logger.QueueNameKey, p.QueueName), slog.Any("error", err))
		return err
	}

	_, err = s.client.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(p.MessageBody),
		QueueUrl:    urlOutput.QueueUrl,
	})
	if err != nil {
		slog.Error("could not sent message", slog.String(logger.QueuUrlKey, *urlOutput.QueueUrl),
			slog.String(logger.EventBodyKey, p.MessageBody), slog.Any("err", err))
		return err
	}

	return nil
}

func (s *SendMessageInput) Validate() error {
	return validation.ValidateStruct(s,
		validation.Field(&s.QueueName, validation.Required),
		validation.Field(&s.MessageID, validation.Required, is.UUID),
		validation.Field(&s.MessageBody, validation.Required, is.JSON),
	)
}
