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
	decision "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/activate_request"
	"google.golang.org/protobuf/encoding/protojson"
)

// Activate returns a flag activation handler
// @Summary Activate a campaign
// @Tags Campaigns
// @Description Activate a campaign for a visitor ID
// @ID activate
// @Accept  json
// @Produce  json
// @Param request body activateBody true "Campaign activation request body"
// @Success 204
// @Failure 400 {object} errorMessage
// @Failure 500 {object} errorMessage
// @Router /activate [post]
func Activate(context *connectors.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		activateRequest := &activate_request.ActivateRequest{}
		data, err := io.ReadAll(req.Body)
		if err != nil {
			utils.WriteServerError(w, err)
			return
		}

		if err := protojson.Unmarshal(data, activateRequest); err != nil {
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

		assignments := map[string]*decision.VisitorCache{
			activateRequest.Cid: {
				VariationID: activateRequest.Vaid,
				Activated:   true,
			},
		}

		errors := make(chan error, 2)
		go func(errors chan error) {
			errors <- context.AssignmentsManager.SaveAssignments(context.EnvID, visitorID, assignments, now, connectors.SaveAssignmentsContext{
				CacheLevel: connectors.Activation,
			})
		}(errors)

		go func(errors chan error) {
			errors <- context.HitsProcessor.TrackHits(
				connectors.TrackingHits{
					CampaignActivations: []*models.CampaignActivation{
						{
							EnvID:       activateRequest.Cid,
							VisitorID:   visitorID,
							CustomerID:  activateRequest.Vid,
							CampaignID:  activateRequest.Caid,
							VariationID: activateRequest.Vaid,
							Timestamp:   now.UnixNano() / 1000000,
						},
					},
				})
		}(errors)

		close(errors)

		for err := range errors {
			if err != nil {
				utils.WriteServerError(w, err)
				return
			}
		}

		// Return a response with a 200 OK status and the campaign payload as an example
		utils.WriteNoContent(w)
	}
}
