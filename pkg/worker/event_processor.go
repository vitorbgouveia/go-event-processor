package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/vitorbgouveia/go-event-processor/internal/models"
	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
	"github.com/vitorbgouveia/go-event-processor/pkg/logger"
)

type (
	eventProcessor struct {
		msgBrotker        aws.MessageBroker
		queueRetryProcess string
		repo              repository.Events
	}

	EventProcessor interface {
		ProcessEvents(ctx context.Context, event *models.EventInput, cancel <-chan struct{}) error
	}

	EventProcessorInput struct {
		MsgBroker         aws.MessageBroker
		QueueRetryProcess string
		Repo              repository.Events
	}
)

var (
	ErrInterruptionProcess   = errors.New("got interruption signal, canceling process")
	ErrCastToDispatchedEvent = errors.New("could not cast raw event to dispatchedEvent")
	ErrEventSentToRetryAgain = errors.New("message has already been sent for retry")
)

func NewDispatchedEventProcessor(d *EventProcessorInput) EventProcessor {
	return &eventProcessor{
		msgBrotker: d.MsgBroker, queueRetryProcess: d.QueueRetryProcess, repo: d.Repo,
	}
}

func (s *eventProcessor) ProcessEvents(ctx context.Context, event *models.EventInput, cancel <-chan struct{}) error {
	result := make(chan error)

	go func(processResult chan<- error, cancelProcess <-chan struct{}) {
		for _, record := range event.Records {
			select {
			case <-cancelProcess:
				if retryErr := s.sendEventToRetry(ctx, record, ErrInterruptionProcess); retryErr != nil {
					processResult <- errors.Join(ErrInterruptionProcess, retryErr)
					return
				}

				processResult <- nil
				return
			default:
				var messageBody models.EventMessageBody
				if err := s.parseMessageBody(record.Body, &messageBody); err != nil {
					if rejectErr := s.msgBrotker.PublishRejectedEvent(ctx, record.Body, err); rejectErr != nil {
						processResult <- errors.Join(err, rejectErr)
						return
					}

					slog.Warn("event rejected, notification sent", slog.String(logger.EventBodyKey, record.Body), slog.Any("err", err))
					processResult <- nil
					return
				}

				if err := s.repo.Insert(repository.EventInsertInput{
					EventId: messageBody.EventId, Context: messageBody.Context, Type: messageBody.Type,
					Tenant: messageBody.Tenant, Data: messageBody.Data,
				}); err != nil {
					if retryErr := s.sendEventToRetry(ctx, record, err); retryErr != nil {
						processResult <- errors.Join(err, retryErr)
						return
					}

					processResult <- nil
					return
				}

				if err := s.msgBrotker.PublishValidEvent(ctx, record.Body); err != nil {
					if retryErr := s.sendEventToRetry(ctx, record, err); retryErr != nil {
						processResult <- errors.Join(err, retryErr)
						return
					}

					processResult <- nil
					return
				}

				slog.Info("event processed", slog.String(logger.EventIdKey, messageBody.EventId),
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

func (s *eventProcessor) parseMessageBody(message string, messageBody *models.EventMessageBody) error {
	if err := json.Unmarshal([]byte(message), &messageBody); err != nil {
		return err
	}

	return messageBody.Validate()
}

func (s *eventProcessor) sendEventToRetry(ctx context.Context, event models.EventRecord, errReason error) error {
	if strings.Contains(event.EventARN, s.queueRetryProcess) {
		return ErrEventSentToRetryAgain
	}

	if err := s.msgBrotker.SendToQueue(ctx, aws.SendToQueueInput{
		QueueName:   s.queueRetryProcess,
		MessageBody: event.Body,
	},
	); err != nil {
		return err
	}

	slog.Warn("event sent to retry", slog.String(logger.MessageIDKey, event.MessageId),
		slog.String(logger.QueueNameKey, s.queueRetryProcess), slog.String(logger.EventBodyKey, event.Body), slog.Any("err", errReason))
	return nil
}
