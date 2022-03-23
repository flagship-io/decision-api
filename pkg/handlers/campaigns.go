package handlers

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/flagship-io/decision-api/internal/apilogic"
	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/flagship-proto/decision_response"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// Campaigns returns a campaigns handler
// @Summary Get all campaigns for the visitor
// @Tags Campaigns
// @Description Get all campaigns value and metadata for a visitor ID and context
// @ID get-campaigns
// @Accept  json
// @Produce  json
// @Param request body campaignsBodySwagger true "Campaigns request body"
// @Success 200 {object} campaignsResponse
// @Failure 400 {object} errorMessage
// @Failure 500 {object} errorMessage
// @Router /campaigns [post]
func Campaigns(context *connectors.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		apilogic.HandleCampaigns(w, req, context, requestCampaignsHandler, utils.NewTracker())
	}
}

func toAccountSettings(e *models.Environment) *decision_response.AccountSettings {
	return &decision_response.AccountSettings{
		EnabledXPC:  e.Common.UseReconciliation,
		Enabled1V1T: e.Common.SingleAssignment,
	}
}

func requestCampaignsHandler(w http.ResponseWriter, handleRequest *handle.Request, err error) {
	if err != nil {
		utils.WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}

	handleRequest.Logger.Infof("formatting campaign response for mode %s", handleRequest.Mode)
	var response = decision_response.DecisionResponseFull{}
	needAggregatedResponse := handleRequest.Mode == "simple" || handleRequest.Mode == "full"
	response.Campaigns = handleRequest.DecisionResponse.Campaigns
	response.VisitorId = handleRequest.DecisionResponse.VisitorId
	response.Extras = handleRequest.DecisionResponse.Extras
	if needAggregatedResponse {
		aggregateFullResponse(&response)
	}

	// extras
	addExtraToResponse(handleRequest, &response, "accountSettings", toAccountSettings(handleRequest.Environment))

	finalMessage := getFilteredMessage(&response, handleRequest.Mode)

	// marshal proto
	var ma protojson.MarshalOptions
	ma.EmitUnpopulated = true
	data, err := ma.Marshal(finalMessage)
	if err != nil {
		utils.WriteServerError(w, fmt.Errorf("error when returning final response %v", err))
	}

	utils.WriteJSONStringOk(w, getSanitizedResponse(string(data)))
}

func aggregateFullResponse(response *decision_response.DecisionResponseFull) {
	response.CampaignsVariation = []*decision_response.CampaignIdVariationId{}
	response.MergedModifications = &structpb.Struct{
		Fields: make(map[string]*structpb.Value),
	}

	for _, campaign := range response.Campaigns {
		response.CampaignsVariation = append(response.CampaignsVariation, &decision_response.CampaignIdVariationId{
			CampaignId:  campaign.Id.GetValue(),
			VariationId: campaign.Variation.Id.GetValue(),
		})

		modification := campaign.Variation.Modifications
		if modification.Value != nil {
			for key, value := range modification.Value.Fields {
				response.MergedModifications.Fields[key] = value
			}
		}
	}
}

func getFilteredMessage(response *decision_response.DecisionResponseFull, mode string) proto.Message {
	switch mode {
	case "full":
		return response
	case "normal":
		msg := decision_response.DecisionResponse{}
		msg.Campaigns = response.Campaigns
		msg.VisitorId = response.VisitorId
		msg.Extras = response.Extras
		return &msg
	case "simple":
		msg := decision_response.DecisionResponseSimple{}
		msg.MergedModifications = response.MergedModifications
		msg.CampaignsVariation = response.CampaignsVariation
		msg.Extras = response.Extras
		return &msg
	}

	return nil
}

// add extra response to header
func addExtraToResponse(handleRequest *handle.Request, response *decision_response.DecisionResponseFull, key string, m proto.Message) {
	if !handleRequest.HasExtra(key) {
		return
	}

	extra, _ := anypb.New(m)
	if response.Extras == nil {
		response.Extras = map[string]*anypb.Any{}
	}
	response.Extras[key] = extra
}

// remove extras information from decision api when empty
func getSanitizedResponse(response string) string {
	m := regexp.MustCompile(`,\s*"extras"\s*:\s*{}`)
	return m.ReplaceAllString(response, "")
}
