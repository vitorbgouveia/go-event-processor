package repository

import (
	"errors"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws"
	"github.com/vitorbgouveia/go-event-processor/pkg/aws/mocks"
)

func TestDispatchedEvents_Insert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	errInsert := errors.New("could not insert event")
	eventInput := EventInsertInput{}
	insertInput := aws.InsertInput{
		TableName:   tableName,
		StructValue: eventInput,
	}

	m := mocks.NewMockPersistence(ctrl)
	gomock.InOrder(
		m.EXPECT().Insert(insertInput).Return(nil),
		m.EXPECT().Insert(insertInput).Return(errInsert),
	)

	repo := NewValidEvents(m)
	err := repo.Insert(eventInput)
	assert.Nil(t, err)

	err = repo.Insert(eventInput)
	assert.EqualError(t, err, errInsert.Error())
}
