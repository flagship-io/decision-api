package apilogic

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	common "github.com/flagship-io/flagship-common"
)

// HandleCampaigns get campaigns from request, add checks and side effect and return response
func HandleCampaigns(w http.ResponseWriter, req *http.Request, decisionContext *connectors.DecisionContext, handleDecision func(http.ResponseWriter, *handle.Request, error), tracker *common.Tracker) {
	handleRequest, err := BuildHandleRequest(req)
	if err != nil {
		utils.WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}

	handleRequest.DecisionContext = decisionContext

	// 1. Get environment info from environment ID & API Key
	tracker.TimeTrack("start get env info from env loader")
	handleRequest.Environment, err = decisionContext.EnvironmentLoader.LoadEnvironment(handleRequest.DecisionContext.EnvID, handleRequest.DecisionContext.APIKey)
	tracker.TimeTrack("end get env info from env loader")

	if err != nil {
		if errors.Is(err, models.ErrEnvironmentNotFound) {
			utils.WriteClientError(w, http.StatusBadRequest, fmt.Sprintf("environment %s not found", handleRequest.DecisionContext.EnvID))
			return
		}
		utils.WriteServerError(w, err)
		return
	}

	// 4. Checks that optional campaign ID exists
	if handleRequest.CampaignID != "" {
		filteredCampaigns := []*common.Campaign{}
		for _, v := range handleRequest.Environment.Campaigns {
			if v.ID == handleRequest.CampaignID || (v.Slug != nil && *v.Slug == handleRequest.CampaignID) {
				filteredCampaigns = append(filteredCampaigns, v)
				break
			}
		}

		if len(filteredCampaigns) == 0 {
			utils.WriteClientError(w, http.StatusBadRequest, fmt.Sprintf("The campaign %s is paused or doesn’t exist. Verify your customId or campaignId.", handleRequest.CampaignID))
			return
		}
		handleRequest.Environment.Campaigns = filteredCampaigns
	}

	// 5. Return panic response is panic mode activated
	if handleRequest.Environment.IsPanic {
		utils.WritePanicResponse(w, handleRequest.DecisionRequest.VisitorId)
		return
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// If sendContext explicitely set to !true or context is empty, return
	if handleRequest.SendContextEvent && len(handleRequest.DecisionRequest.Context) > 0 {
		wg.Add(1)
		go func(wg *sync.WaitGroup, handleRequest *handle.Request, tracker *common.Tracker) {
			defer wg.Done()

			tracker.TimeTrack("start track visitor context")
			SendVisitorContext(handleRequest)
			tracker.TimeTrack("end track visitor context")
		}(wg, handleRequest, tracker)
	}

	go func(wg *sync.WaitGroup, handleRequest *handle.Request, w http.ResponseWriter, tracker *common.Tracker) {
		defer wg.Done()

		tracker.TimeTrack("start compute campaigns request logic")
		err = handle.Decision(handleRequest, tracker)
		if err == nil {
			handleDecision(w, handleRequest, err)
		}
		tracker.TimeTrack("end compute campaigns request logic")
	}(wg, handleRequest, w, tracker)

	wg.Wait()
}
