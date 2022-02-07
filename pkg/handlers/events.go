package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/flagship-io/decision-api/internal/models"
	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/internal/validation"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/golang/protobuf/jsonpb"
	"gitlab.com/canarybay/protobuf/ptypes.git/event_request"
)

func Events(context *models.DecisionContext) func(http.ResponseWriter, *http.Request) {
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
				[]connectors.TrackingHit{&models.VisitorContext{
					EnvID:     context.EnvID,
					VisitorID: eventRequest.VisitorId.Value,
					Context:   contextMap,
					Timestamp: now.UnixNano() / 1000000,
				}},
			)
			if err != nil {
				log.Printf("Error when queuing event request : %v", err)
			}
		default:
			fmt.Printf("Type of event %v not handled", eventRequest.Type)
		}
		utils.WriteNoContent(w)
	}
}
