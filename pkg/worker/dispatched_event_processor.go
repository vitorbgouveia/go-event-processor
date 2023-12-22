package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/vitorbgouveia/go-event-processor/internal/models"
	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
	"github.com/vitorbgouveia/go-event-processor/pkg/logger"
)

type (
	dispatchedEventProcessor struct {
		msgBrotker        aws.MessageBroker
		queueRetryProcess string
		queueDLQProcess   string
		repo              repository.DispatchedEvents
	}

	DispatchedEventProcessor interface {
		ProcessEvents(ctx context.Context, event *models.DispatchedEvent, cancel <-chan struct{}) error
	}

	DispatchedEventProcessorInput struct {
		MsgBroker         aws.MessageBroker
		QueueRetryProcess string
		QueueDLQProcess   string
		Repo              repository.DispatchedEvents
	}
)

var (
	ErrInterruptionProcess   = errors.New("got interruption signal, canceling process")
	ErrCastToDispatchedEvent = errors.New("could not cast raw event to dispatchedEvent")
	ErrEventSentToRetryAgain = errors.New("message has already been sent for retry")
)

func NewDispatchedEventProcessor(d *DispatchedEventProcessorInput) DispatchedEventProcessor {
	return &dispatchedEventProcessor{
		d.MsgBroker, d.QueueRetryProcess, d.QueueDLQProcess, d.Repo,
	}
}

func (s *dispatchedEventProcessor) ProcessEvents(ctx context.Context, event *models.DispatchedEvent, cancel <-chan struct{}) error {
	result := make(chan error)

	go func(processResult chan<- error, cancelProcess <-chan struct{}) {
		for _, record := range event.Records {
			select {
			case <-cancelProcess:
				if retryErr := s.sendEventToRetry(ctx, record, ErrInterruptionProcess); retryErr != nil {
					processResult <- errors.Join(ErrInterruptionProcess, retryErr)
				}

				processResult <- nil
				return
			default:
				var messageBody models.EventMessageBody
				if err := s.parseMessageBody(record.Body, &messageBody); err != nil {
					if retryErr := s.sendEventToRetry(ctx, record, err); retryErr != nil {
						processResult <- errors.Join(err, retryErr)
					}

					processResult <- nil
					return
				}

				if err := s.repo.Insert(ctx, messageBody); err != nil {
					if retryErr := s.sendEventToRetry(ctx, record, err); retryErr != nil {
						processResult <- errors.Join(err, retryErr)
					}

					processResult <- nil
				}

				time.Sleep(3 * time.Second)
				slog.Info("processed dispatched event",
					slog.String(logger.MessageIDKey, record.MessageId), slog.Any(logger.EventBodyKey, messageBody))
				processResult <- nil
			}
		}

		processResult <- nil
	}(result, cancel)

	if err := <-result; err != nil {
		slog.Error("fail to proccess events", slog.Any(logger.EventBodyKey, event), slog.Any("err", err))
		return err
	}

	return nil
}

func (s *dispatchedEventProcessor) parseMessageBody(message string, messageBody *models.EventMessageBody) error {
	if err := json.Unmarshal([]byte(message), &messageBody); err != nil {
		return err
	}

	return messageBody.Validate()
}

func (s *dispatchedEventProcessor) sendEventToRetry(ctx context.Context, event models.EventRecord, errReason error) error {
	if strings.Contains(event.EventARN, s.queueRetryProcess) {
		return ErrEventSentToRetryAgain
	}

	if err := s.msgBrotker.SendMessage(ctx, aws.SendMessageInput{
		QueueName:   s.queueRetryProcess,
		MessageID:   event.MessageId,
		MessageBody: event.Body,
	},
	); err != nil {
		return nil
	}

	slog.Warn("event sent to retry", slog.String(logger.MessageIDKey, event.MessageId),
		slog.String(logger.QueueNameKey, s.queueRetryProcess), slog.String(logger.EventBodyKey, event.Body), slog.Any("err", errReason))
	return nil
}

// func (s *dispatchEventProcessor) sendEventToDeadLetter(ctx context.Context, data interface{}) error {
// 	bBody, err := json.Marshal(data)
// 	if err != nil {
// 		return err
// 	}

// 	if err := s.msgBrotker.SendMessage(ctx,
// 		aws.SendMessageInput{
// 			QueueName:   s.queueDLQProcess,
// 			MessageID:   uuid.NewString(),
// 			MessageBody: string(bBody),
// 		},
// 	); err != nil {
// 		return err
// 	}

// 	slog.Warn("event sent directly to DLQ", slog.String(logger.EventBodyKey, string(bBody)),
// 		slog.String(logger.QueueNameKey, s.queueDLQProcess))
// 	return nil
// }
