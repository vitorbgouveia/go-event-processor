package messagebroker

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/vitorbgouveia/go-event-processor/pkg"
	"go.uber.org/zap"
)

type broker struct {
	logger *zap.SugaredLogger
	sqs    *sqs.Client
}

type SendMessageParams struct {
	QueueName   string
	MessageID   string
	MessageBody string
}

type Broker interface {
	SendMessage(tx context.Context, p SendMessageParams) error
}

func New(logger *zap.SugaredLogger, ctx context.Context) (Broker, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := sqs.NewFromConfig(cfg)

	return &broker{
		sqs: client, logger: logger,
	}, nil
}

func (s *broker) SendMessage(ctx context.Context, p SendMessageParams) error {
	urlOutput, err := s.sqs.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(p.QueueName),
	})
	if err != nil {
		s.logger.Errorw("could not get url queue", zap.String(pkg.QueueNameKey, p.QueueName), zap.Error(err))
		return err
	}

	_, err = s.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(p.MessageBody),
		QueueUrl:    urlOutput.QueueUrl,
	})
	if err != nil {
		s.logger.Errorw("could not sent message", zap.String(pkg.QueuUrlKey, *urlOutput.QueueUrl),
			zap.String(pkg.EventBodyKey, p.MessageBody), zap.Error(err))
		return err
	}

	return nil
}
