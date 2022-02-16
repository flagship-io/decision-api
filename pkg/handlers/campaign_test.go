package handlers

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
)

func TestCampaignAssignment(t *testing.T) {
	url, _ := url.Parse("/campaigns/campaign_1?sendContextEvent=false")
	body := `{"visitor_id": "1234", "context": {}, "trigger_hit": false }`
	w := httptest.NewRecorder()

	req := &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	Campaign(&DecisionContext{
		EnvID:  "env_id_1",
		APIKey: "api_key_id",
	})(w, req)

	resp := w.Result()

	campaign := decision_response.Campaign{}
	err := jsonpb.Unmarshal(resp.Body, &campaign)

	if err != nil {
		log.Fatalln(err)
	}

	if campaign.Id.Value == "campaign_1" {
		assert.Equal(t, "vg_2", campaign.VariationGroupId.Value)
	}

	body = `{"visitor_id": "1234", "context": {}, "trigger_hit": false, "format_response": true }`
	url, _ = url.Parse("/campaigns/image?sendContextEvent=false&format_response=true")
	req.Body = io.NopCloser(strings.NewReader(body))

	Campaign(&DecisionContext{
		EnvID:  "env_id_1",
		APIKey: "api_key_id",
	})(w, req)

	resp = w.Result()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image", resp.Header["Content-Type"])
}
