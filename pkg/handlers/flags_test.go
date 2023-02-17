package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flagship-io/decision-api/internal/test"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	common "github.com/flagship-io/flagship-common"
	"github.com/stretchr/testify/assert"
)

func TestFlags(t *testing.T) {
	url, _ := url.Parse("/flags?sendContextEvent=false")
	body := `{"visitor_id": "1234", "context": {}, "trigger_hit": false }`
	w := httptest.NewRecorder()

	req := &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	Flags(test.CreateMockDecisionContext())(w, req)

	resp := w.Result()

	data := map[string]FlagInfo{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	assert.Nil(t, err)

	flag, ok := data["testString"]

	assert.True(t, ok)
	assert.Equal(t, "string", flag.Value)
	assert.Equal(t, "campaign_1", flag.Metadata.CampaignID)

	// Test empty flags when no campaigns
	w = httptest.NewRecorder()

	req.Body = io.NopCloser(strings.NewReader(body))
	context := test.CreateMockDecisionContext()
	context.EnvironmentLoader.(*environment_loaders.MockLoader).MockedEnvironment.Common.Campaigns = []*common.Campaign{}
	Flags(context)(w, req)

	resp = w.Result()

	data = map[string]FlagInfo{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	assert.Nil(t, err)
	assert.NotNil(t, data)
	assert.Len(t, data, 0)
}
