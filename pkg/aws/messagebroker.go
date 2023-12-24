package aws

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/vitorbgouveia/go-event-processor/pkg/logger"
)

type (
	messagebroker struct {
		sqsClient             sqsiface.SQSAPI
		snsClient             snsiface.SNSAPI
		rejectedEventTopicARN string
		validEventTopicARN    string
	}

	SendToQueueInput struct {
		QueueName   string
		MessageBody string
	}

	CreateTopicsInput struct {
		ValidEventTopic    string
		RejectedEventTopic string
	}

	MessageBroker interface {
		CreateRejectEventTopic(name string) error
		CreateValidEventTopic(name string) error
		SendToQueue(ctx context.Context, p SendToQueueInput) error
		PublishValidEvent(ctx context.Context, message string) error
		PublishRejectedEvent(ctx context.Context, message string, errReason error) error
	}
)

func NewMessageBroker(sess *session.Session) MessageBroker {
	sqsClient := sqs.New(sess)
	snsClient := sns.New(sess)

	return &messagebroker{sqsClient: sqsClient, snsClient: snsClient}
}

func (s *messagebroker) CreateRejectEventTopic(name string) error {
	o, err := s.snsClient.CreateTopic(&sns.CreateTopicInput{
		Name: &name,
	})
	if err != nil {
		return err
	}
	s.rejectedEventTopicARN = *o.TopicArn

	return nil
}

func (s *messagebroker) CreateValidEventTopic(name string) error {
	o, err := s.snsClient.CreateTopic(&sns.CreateTopicInput{
		Name: &name,
	})
	if err != nil {
		return err
	}
	s.validEventTopicARN = *o.TopicArn

	return nil
}

func (s *messagebroker) PublishValidEvent(ctx context.Context, message string) error {
	if _, err := s.snsClient.PublishWithContext(ctx, &sns.PublishInput{
		Message:  &message,
		TopicArn: &s.validEventTopicARN,
	}); err != nil {
		slog.Error("could not publish to topic", slog.String("topic_name", s.validEventTopicARN),
			slog.String(logger.EventBodyKey, message), slog.Any("err", err))
		return err
	}

	return nil
}

func (s *messagebroker) PublishRejectedEvent(ctx context.Context, message string, errReason error) error {
	if _, err := s.snsClient.PublishWithContext(ctx, &sns.PublishInput{
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"reason_reject": {StringValue: aws.String(errReason.Error())}},
		Message:  &message,
		TopicArn: &s.rejectedEventTopicARN,
	}); err != nil {
		slog.Error("could not publish to topic", slog.String("topic_name", s.rejectedEventTopicARN),
			slog.String(logger.EventBodyKey, message), slog.Any("err", err))
		return err
	}

	return nil
}

func (s *messagebroker) SendToQueue(ctx context.Context, i SendToQueueInput) error {
	if err := i.Validate(); err != nil {
		return err
	}

	urlOutput, err := s.sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(i.QueueName),
	})
	if err != nil {
		slog.Error("could not get url queue", slog.String(logger.QueueNameKey, i.QueueName), slog.Any("error", err))
		return err
	}

	_, err = s.sqsClient.SendMessageWithContext(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(i.MessageBody),
		QueueUrl:    urlOutput.QueueUrl,
	})
	if err != nil {
		slog.Error("could not sent message", slog.String(logger.QueuUrlKey, *urlOutput.QueueUrl),
			slog.String(logger.EventBodyKey, i.MessageBody), slog.Any("err", err))
		return err
	}

	return nil
}

func (s *SendToQueueInput) Validate() error {
	return validation.ValidateStruct(s,
		validation.Field(&s.QueueName, validation.Required),
		validation.Field(&s.MessageBody, validation.Required, is.JSON),
	)
}
