package validation

import (
	"testing"

	"github.com/flagship-io/flagship-proto/activate_request"
	"github.com/stretchr/testify/assert"
)

func TestBuildErrorResponse(t *testing.T) {
	errors := map[string]string{"error": "detail"}
	test := BuildErrorResponse(errors)

	assert.Equal(t, test.Errors, errors)
}

func TestCheckErrorBody(t *testing.T) {

	resp := CheckErrorBody("env_id", &activate_request.ActivateRequest{})
	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "Field is mandatory.", resp.Errors["cid"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["vid"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["caid"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["vaid"])

	resp = CheckErrorBody("fake", &activate_request.ActivateRequest{
		Cid: "env_id",
	})
	assert.Equal(t, "Invalid cid.", resp.Errors["cid"])

	resp = CheckErrorBody("env_id", &activate_request.ActivateRequest{
		Cid:  "env_id",
		Vid:  "visitor_id",
		Caid: "campaign_id",
		Vaid: "variation_id",
	})

	assert.Nil(t, resp)
}
