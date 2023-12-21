package worker

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vitorbgouveia/go-event-processor/pkg"
	"github.com/vitorbgouveia/go-event-processor/pkg/contracts"
	"github.com/vitorbgouveia/go-event-processor/pkg/messagebroker"
	"go.uber.org/zap"
)

type dispatchEventProcess struct {
	logger            *zap.SugaredLogger
	msgBrotker        messagebroker.Broker
	queueRetryProcess string
	queueDLQProcess   string
}

type sendEventToRetryParams struct {
	messageID string
	eventARN  string
	eventBody string
}

type DispatchEventProcessor interface {
	ProcessEvents(ctx context.Context, event interface{}, cancel <-chan struct{}) error
}

var (
	ErrInterruptionProcess   = errors.New("got interruption signal, canceling process")
	ErrCastToDispatchedEvent = errors.New("could not cast raw event to dispatchedEvent")
	ErrEventSentToRetryAgain = errors.New("message has already been sent for retry")
)

type DispatchEventProcess struct {
	Logger            *zap.SugaredLogger
	MsgBroker         messagebroker.Broker
	QueueRetryProcess string
	QueueDLQProcess   string
}

func NewDispatchEventProcessor(d *DispatchEventProcess) DispatchEventProcessor {
	return &dispatchEventProcess{
		d.Logger, d.MsgBroker, d.QueueRetryProcess, d.QueueDLQProcess,
	}
}

func (s *dispatchEventProcess) ProcessEvents(ctx context.Context, dataRaw interface{}, cancel <-chan struct{}) error {
	result := make(chan error)

	go func(processResult chan<- error, cancelProcess <-chan struct{}) {
		var event contracts.DispatchedEvent
		if err := s.parseDispatchedEvent(dataRaw, &event); err != nil || len(event.Records) == 0 {
			if dlqErr := s.sendEventToDeadLetter(ctx, dataRaw); dlqErr != nil {
				err := errors.Join(ErrCastToDispatchedEvent, dlqErr)
				s.logger.Errorw("could not cast event type", zap.Any(pkg.EventBodyKey, dataRaw), zap.Error(err))
				processResult <- err
			}

			processResult <- nil
			return
		}

		for _, event := range event.Records {
			select {
			case <-cancelProcess:
				if retryErr := s.sendEventToRetry(ctx, sendEventToRetryParams{
					messageID: event.MessageId, eventBody: event.Body, eventARN: event.EventARN,
				}, ErrInterruptionProcess); retryErr != nil {
					err := errors.Join(ErrInterruptionProcess, retryErr)
					s.logger.Errorw("fail to send event to retry",
						zap.String(pkg.MessageIDKey, event.MessageId), zap.String(pkg.EventBodyKey, event.Body), zap.Error(err))
					processResult <- err
				}

				processResult <- nil
				return
			default:
				var messageBody contracts.EventMessageBody
				if err := s.parseMessageBody(event.Body, &messageBody); err != nil {
					if retryErr := s.sendEventToRetry(ctx, sendEventToRetryParams{
						messageID: event.MessageId, eventBody: event.Body, eventARN: event.EventARN,
					}, err); retryErr != nil {
						err = errors.Join(err, retryErr)
						s.logger.Errorw("fail to send event to retry",
							zap.String(pkg.MessageIDKey, event.MessageId), zap.String(pkg.EventBodyKey, event.Body), zap.Error(err))
						processResult <- err
					}

					processResult <- nil
					return
				}

				time.Sleep(3 * time.Second)
				s.logger.Infow("processed dispatched event",
					zap.String(pkg.MessageIDKey, event.MessageId), zap.Any(pkg.EventBodyKey, messageBody))
			}
		}

		processResult <- nil
	}(result, cancel)

	return <-result
}

func (s *dispatchEventProcess) parseDispatchedEvent(dataRaw interface{}, event *contracts.DispatchedEvent) error {
	b, err := json.Marshal(dataRaw)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &event)
}

func (s *dispatchEventProcess) parseMessageBody(message string, messageBody *contracts.EventMessageBody) error {
	if err := json.Unmarshal([]byte(message), &messageBody); err != nil {
		return err
	}

	return messageBody.Validate()
}

func (s *dispatchEventProcess) sendEventToRetry(ctx context.Context, p sendEventToRetryParams, errReason error) error {
	if strings.Contains(p.eventARN, s.queueRetryProcess) {
		return ErrEventSentToRetryAgain
	}

	bBody, err := json.Marshal(p.eventBody)
	if err != nil {
		return err
	}

	if err := s.msgBrotker.SendMessage(ctx, messagebroker.SendMessageParams{
		QueueName:   s.queueRetryProcess,
		MessageID:   p.messageID,
		MessageBody: string(bBody),
	},
	); err != nil {
		return nil
	}

	s.logger.Warnw("event sent to retry", zap.String(pkg.MessageIDKey, p.messageID),
		zap.String(pkg.QueueNameKey, s.queueRetryProcess), zap.String(pkg.EventBodyKey, string(bBody)), zap.Error(errReason))
	return nil
}

func (s *dispatchEventProcess) sendEventToDeadLetter(ctx context.Context, data interface{}) error {
	bBody, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := s.msgBrotker.SendMessage(ctx,
		messagebroker.SendMessageParams{
			QueueName:   s.queueDLQProcess,
			MessageID:   uuid.NewString(),
			MessageBody: string(bBody),
		},
	); err != nil {
		return err
	}

	s.logger.Warnw("event sent directly to DLQ", zap.String(pkg.EventBodyKey, string(bBody)),
		zap.String(pkg.QueueNameKey, s.queueDLQProcess))
	return nil
}
