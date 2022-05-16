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
	resp := CheckErrorBody(&activate_request.ActivateRequest{})

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "Field is mandatory.", resp.Errors["cid"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["vid"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["caid"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["vaid"])

	resp = CheckErrorBody(&activate_request.ActivateRequest{
		Cid:  "env_id",
		Vid:  "visitor_id",
		Caid: "campaign_id",
		Vaid: "variation_id",
	})

	assert.Nil(t, resp)
}

// CheckErrorBodyMultiple checks a multiple activation request
func TestCheckErrorBodyMultiple(t *testing.T) {
	resp := CheckErrorBodyMultiple(&activate_request.ActivateRequestMultiple{
		Activations: []*activate_request.ActivateRequestMultipleInner{
			{},
		},
	})

	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "Field is mandatory.", resp.Errors["environment_id"])
	assert.Equal(t, "Field is mandatory. It can be set globally or for each specific activation", resp.Errors["visitor_id"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["variation_id"])
	assert.Equal(t, "Field is mandatory.", resp.Errors["variation_group_id"])

	resp = CheckErrorBodyMultiple(&activate_request.ActivateRequestMultiple{
		EnvironmentId: "env_id",
		VisitorId:     "vis_id",
		Activations: []*activate_request.ActivateRequestMultipleInner{
			{
				VariationGroupId: "vg_id",
				VariationId:      "v_id",
			},
		},
	})

	assert.Nil(t, resp)
}
