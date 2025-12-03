package handlers

import (
	"encoding/json"
	"io"
	"log"
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
// @Tags Activate
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
		var activateItems []*activate_request.ActivateRequest

		data, err := io.ReadAll(req.Body)
		if err != nil {
			utils.WriteServerError(w, err)
			return
		}

		// check body unique, if not check body multiple
		activateRequest := &activate_request.ActivateRequest{}
		if err := protojson.Unmarshal(data, activateRequest); err != nil {
			activateRequestBatch := &activate_request.ActivateRequestBatch{}
			if err := protojson.Unmarshal(data, activateRequestBatch); err != nil {
				utils.WriteClientError(w, http.StatusBadRequest, err.Error())
				return
			}

			activateItems = activateRequestBatch.Batch
			for _, activateItem := range activateItems {
				activateItem.Cid = activateRequestBatch.Cid
			}
		} else {
			activateItems = []*activate_request.ActivateRequest{activateRequest}
		}

		// error management & campaign activations
		errorsLength := 0
		errors := make(chan error)
		campaignActivations := []*models.CampaignActivation{}

		for _, activateItem := range activateItems {
			if bodyErr := validation.CheckErrorBody(context.EnvID, activateItem); bodyErr != nil {
				data, _ := json.Marshal(bodyErr)
				utils.WriteClientError(w, http.StatusBadRequest, string(data))
				return
			}

			now := time.Now()

			visitorID := activateItem.Vid
			// If anonymous id is defined
			if activateItem.Aid != nil {
				visitorID = activateItem.Aid.Value
			}

			shouldPersistActivation := false
			environment, err := context.EnvironmentLoader.LoadEnvironment(activateItem.Cid, context.APIKey)
			if err != nil {
				log.Printf("Error when reading existing environment : %v", err)
			} else {
				shouldPersistActivation = environment.Common.CacheEnabled && environment.Common.SingleAssignment
			}

			if shouldPersistActivation {
				existingAssignments, err := context.AssignmentsManager.LoadAssignments(activateItem.Cid, activateItem.Vid)
				if err != nil {
					log.Printf("Error when reading existing assignments : %v", err)
				}

				var vgAssign *decision.VisitorCache
				if existingAssignments != nil {
					vgAssign = existingAssignments.Assignments[activateItem.Caid]
				}

				shouldPersistActivation = vgAssign == nil || !vgAssign.Activated || vgAssign.VariationID != activateItem.Vaid
			}

			if shouldPersistActivation {
				errorsLength++
				go func(activateItem *activate_request.ActivateRequest) {
					var err error = nil
					if context.AssignmentsManager.ShouldSaveAssignments(connectors.SaveAssignmentsContext{
						AssignmentScope: connectors.Activation,
					}) {
						err = context.AssignmentsManager.SaveAssignments(context.EnvID, activateItem.Vid, map[string]*decision.VisitorCache{
							activateItem.Caid: {
								VariationID: activateItem.Vaid,
								Activated:   true,
							},
						}, now)
					}
					errors <- err
				}(activateItem)
			}

			campaignActivations = append(campaignActivations, &models.CampaignActivation{
				EnvID:           activateItem.Cid,
				VisitorID:       visitorID,
				CustomerID:      activateItem.Vid,
				CampaignID:      activateItem.Caid,
				VariationID:     activateItem.Vaid,
				Timestamp:       now.UnixNano() / 1000000,
				PersistActivate: shouldPersistActivation,
				QA:              activateItem.Qa,
				QueueTime:       activateItem.Qt,
			})
		}

		errorsLength++
		go func() {
			errors <- context.HitsProcessor.TrackHits(
				connectors.TrackingHits{
					CampaignActivations: campaignActivations,
				})

		}()

		for i := 0; i < errorsLength; i++ {
			err := <-errors
			if err != nil {
				utils.WriteServerError(w, err)
				return
			}
		}

		// Return a response with a 200 OK status and the campaign payload as an example
		utils.WriteNoContent(w)
	}
}
