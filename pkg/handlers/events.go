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
	"gitlab.com/canarybay/protobuf/ptypes.git/event_request"
)

func Events(context *connectors.DecisionContext) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		now := time.Now()
		eventRequest := &event_request.EventRequest{}
		if err := jsonpb.Unmarshal(req.Body, eventRequest); err != nil {
			utils.WriteServerError(w, err)
			return
		}
		if bodyErr := validation.CheckEventErrorBody(eventRequest); bodyErr != nil {
			data, _ := json.Marshal(bodyErr)
			utils.WriteClientError(w, http.StatusBadRequest, string(data))
			return
		}

		switch eventRequest.Type {
		case event_request.EventRequest_CONTEXT:
			contextMap := map[string]interface{}{}
			for k, v := range eventRequest.Data {
				contextMap[k] = v.AsInterface()
			}
			err := context.HitProcessor.TrackHits(
				connectors.TrackingHits{
					VisitorContext: []*models.VisitorContext{
						{
							EnvID:     context.EnvID,
							VisitorID: eventRequest.VisitorId.Value,
							Context:   contextMap,
							Timestamp: now.UnixNano() / 1000000,
						},
					},
				},
			)
			if err != nil {
				context.Logger.Errorf("error when tracking event request : %v", err)
			}
			context.Logger.Info("event tracked successfully")
		default:
			context.Logger.Errorf("type of event %v not handled", eventRequest.Type)
		}
		utils.WriteNoContent(w)
	}
}
