package models

import (
	"fmt"
	"testing"

	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testCase := []struct {
		description string
		event       EventMessageBody
		err         error
	}{
		{
			description: "should pass when body is valid",
			event: EventMessageBody{
				EventId: uuid.NewString(),
				Context: "Monitoring",
				Type:    "outbound",
				Tenant:  uuid.NewString(),
				Data:    `{"login_counter":20}`,
			},
			err: nil,
		},
		{
			description: "should pass when body is valid",
			event: EventMessageBody{
				EventId: uuid.NewString(),
				Context: "Authorization",
				Type:    "inbound",
				Tenant:  uuid.NewString(),
				Data:    `{"request_payment_id":"234052232322"}`,
			},
			err: nil,
		},
		{
			description: "should pass when body is valid",
			event: EventMessageBody{
				EventId: uuid.NewString(),
				Context: "Validation",
				Type:    "inbound",
				Tenant:  uuid.NewString(),
				Data:    `{"ingestionErr":"error in validate batch order file"}`,
			},
			err: nil,
		},
		{
			description: "should return err tenant not is uuid",
			event: EventMessageBody{
				EventId: uuid.NewString(),
				Context: "Validation",
				Type:    "inbound",
				Tenant:  "123",
				Data:    `{"ingestionErr":"error in validate batch order file"}`,
			},
			err: is.ErrUUID,
		},
		{
			description: "should return err data not is json",
			event: EventMessageBody{
				EventId: uuid.NewString(),
				Context: "Validation",
				Type:    "inbound",
				Tenant:  uuid.NewString(),
				Data:    "ok",
			},
			err: is.ErrJSON,
		},
	}

	for i, tc := range testCase {
		t.Run(fmt.Sprintf("TestValidate_%d_%s", i, tc.description), func(t *testing.T) {
			err := tc.event.Validate()

			if tc.err != nil {
				assert.ErrorContains(t, err, tc.err.Error())
				return
			}

			assert.Nil(t, err)
		})
	}
}
