package validation

import (
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"

	"gitlab.com/canarybay/protobuf/ptypes.git/event_request"

	"github.com/stretchr/testify/assert"
)

func TestBuildEventErrorResponse(t *testing.T) {
	errors := map[string]string{"error": "detail"}
	test := BuildEventErrorResponse(errors)

	assert.Equal(t, test.Errors, errors)
}

func TestCheckEventErrorBody(t *testing.T) {
	resp := CheckEventErrorBody(&event_request.EventRequest{
		VisitorId: &wrappers.StringValue{Value: ""},
		Type:      event_request.EventRequest_NULL,
	})

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "Field is mandatory.", resp.Errors["visitorId"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["type"])

	resp = CheckEventErrorBody(&event_request.EventRequest{
		VisitorId: &wrappers.StringValue{Value: "env_id"},
		Type:      event_request.EventRequest_CONTEXT,
		Data:      map[string]*structpb.Value{},
	})

	assert.Nil(t, resp)
}
