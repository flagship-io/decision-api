package apilogic

import (
	"testing"
	"time"

	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/flagship-common/targeting"
	"github.com/flagship-io/flagship-proto/decision_request"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSendVisitorContext(t *testing.T) {
	now := time.Now()
	hitProcessor := &hits_processors.MockHitProcessor{}
	handleRequest := handle.Request{
		Time: now,
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:   wrapperspb.String("visitor_id"),
			AnonymousId: wrapperspb.String("anonymous_id"),
			Context: map[string]*structpb.Value{
				"key": structpb.NewStringValue("value"),
			},
		},
		DecisionContext: &connectors.DecisionContext{
			EnvID: "env_id",
			Connectors: connectors.Connectors{
				HitsProcessor: hitProcessor,
			},
		},
		FullVisitorContext: &targeting.Context{
			IntegrationProviders: map[string]targeting.ContextMap{
				"mixpanel": map[string]*structpb.Value{
					"key": structpb.NewBoolValue(true),
				},
			},
		},
	}
	SendVisitorContext(&handleRequest)

	assert.Len(t, hitProcessor.TrackedHits.VisitorContext, 2)
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[0].EnvID, "env_id")
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[0].VisitorID, "anonymous_id")
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[0].CustomerID, "visitor_id")
	assert.LessOrEqual(t, hitProcessor.TrackedHits.VisitorContext[0].QueueTime, int64(10))
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[0].Context["key"], "value")

	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[1].EnvID, "env_id")
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[1].VisitorID, "anonymous_id")
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[1].CustomerID, "visitor_id")
	assert.LessOrEqual(t, hitProcessor.TrackedHits.VisitorContext[1].QueueTime, int64(10))
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[1].Context["key"], true)
	assert.Equal(t, hitProcessor.TrackedHits.VisitorContext[1].Partner, "mixpanel")
}
