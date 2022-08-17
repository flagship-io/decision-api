package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/stretchr/testify/assert"
)

func TestActivate(t *testing.T) {
	url, _ := url.Parse("/v2/activate")

	body := `{
		"unknown": "field"
    }`
	w := httptest.NewRecorder()

	req := &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}
	context := utils.CreateMockDecisionContext()
	Activate(context)(w, req)

	resp := w.Result()
	bodyResp, _ := io.ReadAll(resp.Body)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, string(bodyResp), "unknown field")

	body = `{
    }`
	w = httptest.NewRecorder()

	req = &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}
	context = utils.CreateMockDecisionContext()
	Activate(context)(w, req)

	resp = w.Result()
	bodyResp, _ = io.ReadAll(resp.Body)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, string(bodyResp), "Field is mandatory")

	body = `{
		"cid": "env_id",
		"aid": "anonymous_id",
		"vid": "visitor_id",
		"caid": "campaign_id",
		"vaid": "variation_id"
    }`
	w = httptest.NewRecorder()

	req = &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	assignmentManager := assignments_managers.InitMemoryManager()
	hitProcessor := &hits_processors.MockHitProcessor{}
	context.EnvID = "env_id"
	context.AssignmentsManager = assignmentManager
	context.HitsProcessor = hitProcessor
	Activate(context)(w, req)

	resp = w.Result()
	assert.Equal(t, 204, resp.StatusCode)

	cacheVisitor, err := assignmentManager.LoadAssignments("env_id", "visitor_id")
	assert.Nil(t, err)
	assert.Nil(t, cacheVisitor)

	context.EnvironmentLoader.(*environment_loaders.MockLoader).MockedEnvironment.Common.SingleAssignment = true
	context.EnvironmentLoader.(*environment_loaders.MockLoader).MockedEnvironment.Common.CacheEnabled = true

	w = httptest.NewRecorder()
	req.Body = io.NopCloser(strings.NewReader(body))
	Activate(context)(w, req)

	resp = w.Result()
	assert.Equal(t, 204, resp.StatusCode)
	cacheVisitor, err = assignmentManager.LoadAssignments("env_id", "visitor_id")
	assert.Nil(t, err)
	assert.Len(t, cacheVisitor.Assignments, 1)
	assert.Equal(t, "variation_id", cacheVisitor.Assignments["campaign_id"].VariationID)
	assert.True(t, cacheVisitor.Assignments["campaign_id"].Activated)

	assert.Len(t, hitProcessor.TrackedHits.CampaignActivations, 1)
	assert.Equal(t, "variation_id", hitProcessor.TrackedHits.CampaignActivations[0].VariationID)
	assert.Equal(t, "campaign_id", hitProcessor.TrackedHits.CampaignActivations[0].CampaignID)
	assert.Equal(t, "env_id", hitProcessor.TrackedHits.CampaignActivations[0].EnvID)
	assert.Equal(t, "visitor_id", hitProcessor.TrackedHits.CampaignActivations[0].CustomerID)
	assert.Equal(t, "anonymous_id", hitProcessor.TrackedHits.CampaignActivations[0].VisitorID)
	assert.True(t, hitProcessor.TrackedHits.CampaignActivations[0].PersistActivate)
}
