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
	"github.com/flagship-io/flagship-common/targeting"
	"google.golang.org/protobuf/types/known/structpb"
)

// HandleCampaigns get campaigns from request, add checks and side effect and return response
func HandleCampaigns(w http.ResponseWriter, req *http.Request, decisionContext *connectors.DecisionContext, handleDecision func(http.ResponseWriter, *handle.Request, error), tracker *common.Tracker) {
	handleRequest, err := BuildHandleRequest(req)
	if err != nil {
		utils.WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}

	handleRequest.DecisionContext = decisionContext
	handleRequest.Logger = decisionContext.Logger

	// 1. Get environment info from environment ID & API Key
	tracker.TimeTrack("start get env info from env loader")
	handleRequest.Logger.Infof("loading environment id: %s", handleRequest.DecisionContext.EnvID)
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

	// 2. Checks that optional campaign ID exists
	if handleRequest.CampaignID != "" {
		handleRequest.Logger.Infof("checking campaign exists with id: %s", handleRequest.CampaignID)
		filteredCampaigns := []*common.Campaign{}
		for _, v := range handleRequest.Environment.Common.Campaigns {
			if v.ID == handleRequest.CampaignID || (v.Slug != nil && *v.Slug == handleRequest.CampaignID) {
				filteredCampaigns = append(filteredCampaigns, v)
				break
			}
		}

		if len(filteredCampaigns) == 0 {
			utils.WriteClientError(w, http.StatusBadRequest, fmt.Sprintf("The campaign %s is paused or doesn’t exist. Verify your customId or campaignId.", handleRequest.CampaignID))
			return
		}
		handleRequest.Environment.Common.Campaigns = filteredCampaigns
	}

	// 3. Return panic response is panic mode activated
	if handleRequest.Environment.Common.IsPanic {
		utils.WritePanicResponse(w, handleRequest.DecisionRequest.VisitorId)
		return
	}

	// 4 Get context keys from integration service (if needed)
	if handleRequest.Environment.HasIntegrations {
		tracker.TimeTrack("start get visitor context from integration service")
		decisionContext.Logger.Info("filling integration visitor context")
		err := fillVisitorContext(handleRequest)
		if err != nil {
			handleRequest.DecisionContext.Logger.Warnf("error occured when getting integration visitor context: %v", err)
		}
		tracker.TimeTrack("end get visitor context from integration service")
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// If sendContext explicitely set to !true or context is empty, return
	if handleRequest.SendContextEvent && len(handleRequest.DecisionRequest.Context) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			tracker.TimeTrack("start track visitor context")
			handleRequest.Logger.Info("sending visitor context to hits processor")
			SendVisitorContext(handleRequest)
			tracker.TimeTrack("end track visitor context")
		}()
	}

	go func() {
		defer wg.Done()

		tracker.TimeTrack("start compute campaigns request logic")
		handleRequest.Logger.Info("computing campaigns decision for visitor ID and context")
		err = handle.Decision(handleRequest, tracker)
		if err == nil {
			handleDecision(w, handleRequest, err)
		}
		tracker.TimeTrack("end compute campaigns request logic")
	}()

	wg.Wait()
}

func fillVisitorContext(request *handle.Request) error {
	data, err := utils.FetchVisitorData(request.DecisionContext.EnvID, request.DecisionRequest.VisitorId.Value)
	if err != nil {
		return err
	}

	request.Logger.Infof("got integration context for %d providers", len(data))
	for _, row := range data {
		if _, ok := request.FullVisitorContext.IntegrationProviders[row.Partner]; !ok {
			request.FullVisitorContext.IntegrationProviders[row.Partner] = targeting.ContextMap{}
		}

		request.FullVisitorContext.IntegrationProviders[row.Partner][row.Segment] = structpb.NewStringValue(row.Value)
	}

	return nil
}
