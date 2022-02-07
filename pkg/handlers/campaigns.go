package handlers

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/flagship-io/decision-api/internal/apilogic"
	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/internal/models"
	"github.com/flagship-io/decision-api/internal/utils"
	common "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/decision_response"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func Campaigns(context *models.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		apilogic.HandleCampaigns(w, req, context, requestCampaignsHandler, utils.NewTracker())
	}
}

func toAccountSettings(e *common.Environment) *decision_response.AccountSettings {
	return &decision_response.AccountSettings{
		EnabledXPC:  e.UseReconciliation,
		Enabled1V1T: e.SingleAssignment,
	}
}

func requestCampaignsHandler(w http.ResponseWriter, handleRequest *handle.Request, err error) {
	if err != nil {
		utils.WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}

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
