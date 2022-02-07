package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/flagship-io/decision-api/internal/models"
	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/internal/validation"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/golang/protobuf/jsonpb"
	"gitlab.com/canarybay/protobuf/ptypes.git/activate_request"
)

func Activate(context *models.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		activateRequest := &activate_request.ActivateRequest{}
		if err := jsonpb.Unmarshal(req.Body, activateRequest); err != nil {
			utils.WriteClientError(w, http.StatusBadRequest, err.Error())
			return
		}

		if bodyErr := validation.CheckErrorBody(activateRequest); bodyErr != nil {
			data, _ := json.Marshal(bodyErr)
			utils.WriteClientError(w, http.StatusBadRequest, string(data))
			return
		}

		now := time.Now()

		visitorID := activateRequest.Vid
		// If anonymous id is defined
		if activateRequest.Aid != nil {
			visitorID = activateRequest.Aid.Value
		}

		context.HitProcessor.TrackHits([]connectors.TrackingHit{
			&models.CampaignActivation{
				EnvID:       activateRequest.Cid,
				VisitorID:   visitorID,
				CustomerID:  activateRequest.Vid,
				CampaignID:  activateRequest.Caid,
				VariationID: activateRequest.Vaid,
				Timestamp:   now.UnixNano() / 1000000,
			},
		})

		// Return a response with a 200 OK status and the campaign payload as an example
		utils.WriteNoContent(w)
	}
}
