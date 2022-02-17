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

func TestCampaignsAssignment(t *testing.T) {
	url, _ := url.Parse("/campaigns?mode=full&sendContextEvent=false")
	body := `{"visitor_id": "1234", "context": {}, "trigger_hit": false }`
	w := httptest.NewRecorder()

	req := &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	Campaigns(utils.CreateMockDecisionContext())(w, req)

	resp := w.Result()

	decisionResponse := decision_response.DecisionResponseFull{}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	err = protojson.Unmarshal(data, &decisionResponse)

	if err != nil {
		log.Fatalln("error unmarshaling campaigns.", err)
	}

	if decisionResponse.VisitorId.Value != "1234" {
		t.Errorf("Wrong visitor ID : %v instead of 1234", decisionResponse.VisitorId.Value)
	}

	// test campaign variations
	for _, c := range decisionResponse.Campaigns {
		if c.Id.Value == "campaign_1" {
			assert.Equal(t, "vg_2", c.VariationGroupId.Value)
		} else if c.Id.Value == "image" {
			assert.Equal(t, "vg_1", c.VariationGroupId.Value)
		}
	}

	// test merged modifications
	assert.Equal(t, decisionResponse.MergedModifications.Fields["testString"].AsInterface(), "string")
	assert.Equal(t, decisionResponse.MergedModifications.Fields["testBool"].AsInterface(), true)
	assert.Equal(t, decisionResponse.MergedModifications.Fields["testNumber"].AsInterface(), 11.)
	assert.Equal(t, decisionResponse.MergedModifications.Fields["testWhatever"].AsInterface(), []interface{}{"a", 1.})
}
