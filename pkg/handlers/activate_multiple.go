package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/internal/validation"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/golang/protobuf/jsonpb"
	"gitlab.com/canarybay/protobuf/ptypes.git/activate_request"
)

func ActivateMultiple(context *connectors.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		now := time.Now()
		envID := context.EnvID

		activateRequest := &activate_request.ActivateRequestMultiple{}
		if err := jsonpb.Unmarshal(req.Body, activateRequest); err != nil {
			utils.WriteClientError(w, http.StatusBadRequest, err.Error())
			return
		}

		activateRequest.EnvironmentId = envID
		if bodyErr := validation.CheckErrorBodyMultiple(activateRequest); bodyErr != nil {
			data, _ := json.Marshal(bodyErr)
			utils.WriteClientError(w, http.StatusBadRequest, string(data))
			return
		}

		campaignActivations := []*models.CampaignActivation{}
		for _, a := range activateRequest.Activations {
			visitorID := activateRequest.VisitorId
			if a.VisitorId != "" {
				visitorID = a.VisitorId
			}
			campaignActivations = append(campaignActivations, &models.CampaignActivation{
				EnvID:       activateRequest.EnvironmentId,
				VisitorID:   visitorID,
				CampaignID:  a.VariationGroupId,
				VariationID: a.VariationId,
				Timestamp:   now.UnixNano() / 1000000,
			})
		}

		context.HitProcessor.TrackHits(connectors.TrackingHits{
			CampaignActivations: campaignActivations,
		})

		// Return a response with a 204 OK status and an empty payload
		utils.WriteNoContent(w)
	}
}
