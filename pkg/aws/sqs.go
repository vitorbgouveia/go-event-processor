package aws

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/vitorbgouveia/go-event-processor/pkg/logger"
)

type (
	messagebroker struct {
		client *sqs.Client
	}

	SendMessageInput struct {
		QueueName   string
		MessageID   string
		MessageBody string
	}

	MessageBroker interface {
		SendMessage(ctx context.Context, p SendMessageInput) error
	}
)

func NewMessageBroker(cfg aws.Config) MessageBroker {
	client := sqs.NewFromConfig(cfg)
	return &messagebroker{client}
}

func (s *messagebroker) SendMessage(ctx context.Context, p SendMessageInput) error {
	urlOutput, err := s.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(p.QueueName),
	})
	if err != nil {
		slog.Error("could not get url queue", slog.String(logger.QueueNameKey, p.QueueName), slog.Any("error", err))
		return err
	}

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
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
