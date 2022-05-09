package handlers

import (
	"net/http"

	"github.com/flagship-io/decision-api/internal/apilogic"
	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/flagship-proto/decision_response"
)

// FlagMetadata represents the metadata informations about a flag key
type FlagMetadata struct {
	CampaignID       string `json:"campaignId"`
	VariationGroupID string `json:"variationGroupId"`
	VariationID      string `json:"variationId"`
}

// FlagInfo represents the informations about a flag key
type FlagInfo struct {
	Value    interface{}  `json:"value"`
	Metadata FlagMetadata `json:"metadata"`
}

// Flags returns a flags handler
// @Summary Get all flags
// @Tags Flags
// @Description Get all flags value and metadata for a visitor ID and context
// @ID get-flags
// @Accept  json
// @Produce  json
// @Param request body campaignsBodySwagger true "Flag request body"
// @Success 200 {object} map[string]FlagInfo{}
// @Failure 400 {object} errorMessage
// @Failure 500 {object} errorMessage
// @Router /flags [post]
func Flags(context *connectors.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		apilogic.HandleCampaigns(w, req, context, requestFlagsHandler, utils.NewTracker())
	}
}

func requestFlagsHandler(w http.ResponseWriter, handleRequest *handle.Request, err error) {
	if err != nil {
		utils.WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}

	sendFlagsResponse(w, handleRequest.DecisionResponse)
}

func sendFlagsResponse(w http.ResponseWriter, decisionResponse *decision_response.DecisionResponse) {
	flagInfos := make(map[string]*FlagInfo)

	for _, c := range decisionResponse.Campaigns {
		if c.GetVariation() != nil && c.GetVariation().GetModifications() != nil && c.GetVariation().GetModifications().GetValue() != nil && c.GetVariation().GetModifications().GetValue().GetFields() != nil {
			for k, v := range c.GetVariation().GetModifications().GetValue().GetFields() {
				flagInfos[k] = &FlagInfo{
					Value: v,
					Metadata: FlagMetadata{
						CampaignID:       c.GetId().Value,
						VariationGroupID: c.GetVariationGroupId().Value,
						VariationID:      c.GetVariation().GetId().Value,
					},
				}
			}
		}
	}

	utils.WriteJSONOk(w, flagInfos)
}
