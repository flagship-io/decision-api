package handlers

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCampaign(t *testing.T) {
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

	body = `{"visitor_id": "1234", "context": {}, "trigger_hit": false, "format_response": true }`
	url, _ = url.Parse("/v2/campaigns/campaign_2")
	req.URL = url
	req.Body = io.NopCloser(strings.NewReader(body))
	w = httptest.NewRecorder()

	Campaign(utils.CreateMockDecisionContext())(w, req)

	resp = w.Result()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, 0, len(bodyBytes))
}

func TestSendSingleFormatResponse(t *testing.T) {
	logger := logger.New("debug", logger.FORMAT_TEXT, "test")
	// json format
	w := httptest.NewRecorder()
	campaign := &decision_response.Campaign{
		Variation: &decision_response.Variation{
			Modifications: &decision_response.Modifications{
				Type: decision_response.ModificationsType_HTML,
				Value: &structpb.Struct{
					Fields: map[string]*structpb.Value{},
				},
			},
		},
	}
	sendSingleFormatResponse(w, campaign, logger)

	resp := w.Result()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"variation":`)

	// other modification type returns json format
	w = httptest.NewRecorder()
	campaign.Variation.Modifications.Type = decision_response.ModificationsType_JSON
	sendSingleFormatResponse(w, campaign, logger)

	resp = w.Result()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	body, _ = io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"variation":`)

	// html type with single fields returns text/html with html value
	w = httptest.NewRecorder()
	campaign.Variation.Modifications.Type = decision_response.ModificationsType_HTML
	campaign.Variation.Modifications.Value.Fields["key"] = structpb.NewStringValue("value")
	sendSingleFormatResponse(w, campaign, logger)

	resp = w.Result()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "value", string(body))

	// text type with single fields returns text/plain withs stringified bool value
	w = httptest.NewRecorder()
	campaign.Variation.Modifications.Type = decision_response.ModificationsType_TEXT
	campaign.Variation.Modifications.Value.Fields["key"] = structpb.NewBoolValue(true)
	sendSingleFormatResponse(w, campaign, logger)

	resp = w.Result()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "true", string(body))

	// text type with single fields returns text/plain withs stringified number value
	w = httptest.NewRecorder()
	campaign.Variation.Modifications.Value.Fields["key"] = structpb.NewNumberValue(20.5)
	sendSingleFormatResponse(w, campaign, logger)

	resp = w.Result()
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "20.5", string(body))
}
