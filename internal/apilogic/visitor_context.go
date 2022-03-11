package apilogic

import (
	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
)

// SendVisitorContext sends a pubsub event to handle visitor context
func SendVisitorContext(handleRequest *handle.Request) {
	contextMap := map[string]interface{}{}
	for k, v := range handleRequest.DecisionRequest.Context {
		contextMap[k] = v.AsInterface()
	}

	visitorID := handleRequest.DecisionRequest.VisitorId.GetValue()
	// If anonymous id is defined
	if handleRequest.DecisionRequest.AnonymousId != nil {
		visitorID = handleRequest.DecisionRequest.AnonymousId.GetValue()
	}

	contexts := []*models.VisitorContext{{
		EnvID:      handleRequest.DecisionContext.EnvID,
		VisitorID:  visitorID,
		CustomerID: handleRequest.DecisionRequest.VisitorId.GetValue(),
		Context:    contextMap,
		Timestamp:  handleRequest.Time.UnixNano() / 1000000,
	}}

	for partner, context := range handleRequest.FullVisitorContext.IntegrationProviders {
		contextMap := map[string]interface{}{}
		for k, v := range context {
			contextMap[k] = v.AsInterface()
		}
		contexts = append(contexts, &models.VisitorContext{
			EnvID:      handleRequest.DecisionContext.EnvID,
			VisitorID:  visitorID,
			Partner:    partner,
			CustomerID: handleRequest.DecisionRequest.VisitorId.GetValue(),
			Context:    contextMap,
			Timestamp:  handleRequest.Time.UnixNano() / 1000000,
		})
	}
	err := handleRequest.DecisionContext.HitsProcessor.TrackHits(connectors.TrackingHits{VisitorContext: contexts})
	if err != nil {
		handleRequest.Logger.Errorf("Error on queuing visitor context : %v", err)
	}
}
