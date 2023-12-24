package pkg

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vitorbgouveia/go-event-processor/internal/models"
	"github.com/vitorbgouveia/go-event-processor/internal/repository"
	repomocks "github.com/vitorbgouveia/go-event-processor/internal/repository/mocks"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
	awsmocks "github.com/vitorbgouveia/go-event-processor/pkg/aws/mocks"
)

var (
	retryErr  = errors.New("fail to insert event")
	insertErr = errors.New("fail to insert event")
)

func TestHandle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	messageID := uuid.NewString()
	tenant := uuid.NewString()
	eventId := uuid.NewString()
	validBody := fmt.Sprintf(`{"event_id": "%s", "context": "monitoring", "type": "inbound", "tenant": "%s", "data": "{\"payment\": 2000}"}`, eventId, tenant)

	mockMsgBrotker := awsmocks.NewMockMessageBroker(ctrl)
	mockMsgBrotker.EXPECT().SendToQueue(context.Background(), aws.SendToQueueInput{
		QueueName: "retry_queue", MessageBody: validBody,
	}).Return(retryErr)

	mockMsgBrotker.EXPECT().PublishValidEvent(context.Background(), validBody).Return(nil).MaxTimes(1)

	mockRepo := repomocks.NewMockEvents(ctrl)
	gomock.InOrder(
		mockRepo.EXPECT().Insert(repository.EventInsertInput{
			EventId: eventId, Context: "monitoring", Type: "inbound", Tenant: tenant, Data: `{"payment": 2000}`,
		}).Return(nil),
		mockRepo.EXPECT().Insert(repository.EventInsertInput{
			EventId: eventId, Context: "monitoring", Type: "inbound", Tenant: tenant, Data: `{"payment": 2000}`,
		}).Return(insertErr),
	)

	lh := NewLambdaHandler(LambdaHandlerInput{
		QueueRetryProcess: "retry_queue", QueueDLQProcess: "dlq_queue",
		MsgBroker: mockMsgBrotker, DispatchedEventRepo: mockRepo,
	})
	event := &models.EventInput{
		Records: []models.EventRecord{
			{
				MessageId: messageID,
				Body:      validBody, EventARN: "any",
			},
		},
	}

	err := lh.Handle(context.Background(), event)
	assert.Nil(t, err)

	err = lh.Handle(context.Background(), event)
	assert.ErrorContains(t, err, insertErr.Error())
	assert.ErrorContains(t, err, retryErr.Error())
}
