package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/internal/validation"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/flagship-proto/activate_request"
	"google.golang.org/protobuf/encoding/protojson"
)

func ActivateMultiple(context *connectors.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		now := time.Now()
		envID := context.EnvID

		activateRequest := &activate_request.ActivateRequestMultiple{}
		data, err := io.ReadAll(req.Body)
		if err != nil {
			utils.WriteServerError(w, err)
			return
		}

		if err := protojson.Unmarshal(data, activateRequest); err != nil {
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

		err = context.HitsProcessor.TrackHits(connectors.TrackingHits{
			CampaignActivations: campaignActivations,
		})

		if err != nil {
			utils.WriteServerError(w, err)
			return
		}

		// Return a response with a 204 OK status and an empty payload
		utils.WriteNoContent(w)
	}
}
