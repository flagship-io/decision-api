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
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCampaigns(t *testing.T) {
	// Full mode, sendContextEvent false
	url, _ := url.Parse("/campaigns?mode=full&sendContextEvent=false")
	body := `{"visitor_id": "1234", "context": {"key": "value"}, "trigger_hit": false }`
	w := httptest.NewRecorder()

	req := &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	decisionContext := utils.CreateMockDecisionContext()
	Campaigns(decisionContext)(w, req)

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

	// normal mode, send context events true, extras
	url, _ = url.Parse("/campaigns?mode=normal&extras=accountSettings")
	w = httptest.NewRecorder()
	req = &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	Campaigns(decisionContext)(w, req)
	resp = w.Result()

	decisionResponseNormal := decision_response.DecisionResponse{}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	err = protojson.Unmarshal(data, &decisionResponseNormal)
	assert.Nil(t, err)

	hitsProcessors := decisionContext.Connectors.HitsProcessor.(*hits_processors.MockHitProcessor)
	assert.Equal(t, 1, len(hitsProcessors.TrackedHits.VisitorContext))
	assert.Equal(t, 2, len(decisionResponseNormal.Campaigns))
	assert.Equal(t, 1, len(decisionResponseNormal.Extras))
	assert.NotNil(t, 0, decisionResponseNormal.Extras["accountSettings"])

	// simple mode
	url, _ = url.Parse("/campaigns?mode=simple")
	w = httptest.NewRecorder()
	req = &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	Campaigns(decisionContext)(w, req)
	resp = w.Result()

	decisionResponseSimple := decision_response.DecisionResponseSimple{}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	err = protojson.Unmarshal(data, &decisionResponseSimple)
	assert.Nil(t, err)

	assert.Equal(t, 5, len(decisionResponseSimple.MergedModifications.Fields))
	assert.Equal(t, 2, len(decisionResponseSimple.CampaignsVariation))
}
