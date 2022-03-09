package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flagship-io/decision-api/internal/utils"
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

	Flags(utils.CreateMockDecisionContext())(w, req)

	resp := w.Result()

	data := map[string]FlagInfo{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	assert.Nil(t, err)

	flag, ok := data["testString"]

	assert.True(t, ok)
	assert.Equal(t, "string", flag.Value)
	assert.Equal(t, "campaign_1", flag.Metadata.CampaignID)
}
