package validation

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/flagship-io/flagship-proto/event_request"

	"github.com/stretchr/testify/assert"
)

func TestBuildEventErrorResponse(t *testing.T) {
	errors := map[string]string{"error": "detail"}
	test := BuildEventErrorResponse(errors)

	assert.Equal(t, test.Errors, errors)
}

func TestCheckEventErrorBody(t *testing.T) {
	resp := CheckEventErrorBody(&event_request.EventRequest{
		VisitorId: &wrapperspb.StringValue{Value: ""},
		Type:      event_request.EventRequest_NULL,
	})

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "Field is mandatory.", resp.Errors["visitorId"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["type"])

	resp = CheckEventErrorBody(&event_request.EventRequest{
		VisitorId: &wrapperspb.StringValue{Value: "env_id"},
		Type:      event_request.EventRequest_CONTEXT,
		Data:      map[string]*structpb.Value{},
	})

	assert.Nil(t, resp)
}
