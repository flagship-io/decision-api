package handlers

import (
	"net/http"
	"strconv"

	"github.com/flagship-io/decision-api/internal/apilogic"
	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/flagship-proto/decision_response"
	protoStruct "github.com/golang/protobuf/ptypes/struct"

	"github.com/golang/protobuf/jsonpb"
)

func Campaign(context *connectors.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		apilogic.HandleCampaigns(w, req, context, requestCampaignHandler, utils.NewTracker())
	}
}

func requestCampaignHandler(w http.ResponseWriter, handleRequest *handle.Request, err error) {
	if err != nil {
		utils.WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}

	if handleRequest.DecisionRequest.GetFormatResponse() != nil && handleRequest.DecisionRequest.GetFormatResponse().GetValue() {
		sendSingleFormatResponse(w, handleRequest.DecisionResponse.Campaigns[0])
		return
	}

	if len(handleRequest.DecisionResponse.Campaigns) == 0 {
		utils.WriteNoContent(w)
		return
	}

	sendSingleResponse(w, handleRequest.DecisionResponse.Campaigns[0])
}

func sendSingleResponse(w http.ResponseWriter, campaignDecisionResponse *decision_response.Campaign) {
	ma := jsonpb.Marshaler{EmitDefaults: true}

	err := ma.Marshal(w, campaignDecisionResponse)

	if err != nil {
		utils.WriteServerError(w, err)
		return
	}
}

func sendSingleFormatResponse(w http.ResponseWriter, campaignDecisionResponse *decision_response.Campaign) {
	contentType := "application/json"
	switch campaignDecisionResponse.GetVariation().GetModifications().GetType() {
	case decision_response.ModificationsType_IMAGE:
		contentType = "image"
	case decision_response.ModificationsType_TEXT:
		contentType = "text/plain"
	case decision_response.ModificationsType_HTML:
		contentType = "text/html"
	default:
		sendSingleResponse(w, campaignDecisionResponse)
		return
	}

	fields := campaignDecisionResponse.GetVariation().GetModifications().GetValue().GetFields()
	var value *protoStruct.Value
	for _, v := range fields {
		value = v
	}

	if value == nil {
		sendSingleResponse(w, campaignDecisionResponse)
		return
	}

	dataValue := ""
	switch value.Kind.(type) {
	case (*protoStruct.Value_StringValue):
		dataValue = value.GetStringValue()
	case (*protoStruct.Value_BoolValue):
		dataValue = strconv.FormatBool(value.GetBoolValue())
	case (*protoStruct.Value_NumberValue):
		dataValue = strconv.FormatFloat(value.GetNumberValue(), 'E', -1, 64)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dataValue))
	w.Header().Add("Content-Type", contentType)
	w.Header().Add("Access-Control-Allow-Origin", "*")
}
