package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type (
	dynamodbClientMock struct {
		dynamodbiface.DynamoDBAPI
	}
)

func (s *dynamodbClientMock) PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return nil, nil
}

func TestPersistence_Insert_config_Err(t *testing.T) {
	t.Parallel()

	type testInput struct {
		AccountId string  `json:"account_id"`
		Value     float32 `json:""`
	}

	p := NewPersistence(session.Must(session.NewSessionWithOptions(
		session.Options{
			SharedConfigState: session.SharedConfigEnable,
		},
	)))
	err := p.Insert(InsertInput{
		TableName: "___event_table___", StructValue: testInput{
			AccountId: uuid.NewString(), Value: 2230,
		},
	})
	assert.ErrorIs(t, err, aws.ErrMissingRegion)
}

func TestPersistence_Insert(t *testing.T) {
	p := &persistence{
		client: &dynamodbClientMock{},
	}

	type testInput struct {
		AccountId string  `json:"account_id"`
		Value     float32 `json:""`
	}
	validInput := testInput{
		AccountId: uuid.NewString(),
		Value:     12000,
	}

	testCase := []struct {
		description string
		input       InsertInput
		err         error
	}{
		{
			description: "should return error when table name is empty",
			input: InsertInput{
				TableName:   "",
				StructValue: validInput,
			},
			err: ErrInvalidTableName,
		},
		{
			description: "should return error when structValue is invalid",
			input: InsertInput{
				TableName:   "event_name",
				StructValue: "invalid_value",
			},
			err: ErrInvalidInsertInput,
		},
		{
			description: "should return success when structValue and table name is valid",
			input: InsertInput{
				TableName:   "event_name",
				StructValue: validInput,
			},
			err: nil,
		},
	}

	for i, tc := range testCase {
		t.Run(fmt.Sprintf("TestPersistence_Insert_%d_%s", i, tc.description), func(t *testing.T) {
			err := p.Insert(tc.input)

			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}

			assert.Nil(t, err)
		})
	}
}
