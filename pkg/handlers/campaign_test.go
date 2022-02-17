package handlers

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCampaignAssignment(t *testing.T) {
	url, _ := url.Parse("/v2/campaigns/campaign_1?sendContextEvent=false")
	body := `{"visitor_id": "1234", "context": {}, "trigger_hit": false }`
	w := httptest.NewRecorder()

	req := &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	Campaign(utils.CreateMockDecisionContext())(w, req)

	resp := w.Result()

	campaign := decision_response.Campaign{}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	err = protojson.Unmarshal(data, &campaign)

	if err != nil {
		log.Fatalln(err)
	}

	if campaign.Id.Value == "campaign_1" {
		assert.Equal(t, "vg_2", campaign.VariationGroupId.Value)
	}

	body = `{"visitor_id": "1234", "context": {}, "trigger_hit": false, "format_response": true }`
	url, _ = url.Parse("/v2/campaigns/image?sendContextEvent=false&format_response=true")
	req.URL = url
	req.Body = io.NopCloser(strings.NewReader(body))
	w = httptest.NewRecorder()

	Campaign(utils.CreateMockDecisionContext())(w, req)

	resp = w.Result()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image", resp.Header.Get("Content-Type"))
}
