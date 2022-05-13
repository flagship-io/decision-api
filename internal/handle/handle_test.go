package handle

import (
	"testing"

	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/flagship-common/targeting"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/flagship-io/flagship-proto/decision_request"

	common "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/stretchr/testify/assert"
)

var vgTest *common.VariationGroup = &common.VariationGroup{
	Campaign: &common.Campaign{
		ID: "testCampaignId",
	},
	ID: "testId",
	Variations: []*common.Variation{
		{
			Modifications: &decision_response.Modifications{
				Value: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"test": {
							Kind: &structpb.Value_StringValue{
								StringValue: "youpi",
							},
						},
					},
				},
			},
			Allocation: 50,
		}, {
			Modifications: &decision_response.Modifications{
				Value: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"test2": {
							Kind: &structpb.Value_StringValue{
								StringValue: "youpi",
							},
						},
					},
				},
			},
			Allocation: 100,
		},
	},
}

func TestGetCampaignResponse(t *testing.T) {
	respTestNotAllKeys := buildCampaignResponse(vgTest, vgTest.Variations[0], false)

	nbKeys := len(respTestNotAllKeys.Variation.Modifications.Value.Fields)
	if nbKeys != 1 {
		t.Errorf("Expected %v modif value because of exposeAllKeys true. Got %v", 1, nbKeys)
	}

	respTest := buildCampaignResponse(vgTest, vgTest.Variations[0], true)

	if respTest.Id.Value != vgTest.Campaign.ID {
		t.Errorf("Expected campaign ID %v. Got %v", vgTest.Campaign.ID, respTest.Id)
	}

	nbKeys = len(respTest.Variation.Modifications.Value.Fields)
	if nbKeys != 2 {
		t.Errorf("Expected %v modif value because of exposeAllKeys true. Got %v", 2, nbKeys)
	}
}

func TestDecision(t *testing.T) {
	err := Decision(&Request{}, nil)
	assert.NotNil(t, err)

	visID1 := "vis_id"
	clientID := "client_id"

	campaigns := []*common.Campaign{
		utils.CreateABCampaignMock(
			"campaign1",
			"vg1",
			utils.CreateAllUsersTargetingMock(),
			utils.CreateModification("key", "value", decision_response.ModificationsType_FLAG),
		),
	}
	clientInfos := common.Environment{
		ID:        clientID,
		Campaigns: campaigns,
	}
	hitsProcessor := &hits_processors.MockHitProcessor{}
	handleRequest := Request{
		DecisionContext: &connectors.DecisionContext{
			EnvID: clientID,
			Connectors: connectors.Connectors{
				HitsProcessor: hitsProcessor,
			},
		},
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:  &wrapperspb.StringValue{Value: visID1},
			TriggerHit: wrapperspb.Bool(true),
			Context:    map[string]*structpb.Value{},
		},
		FullVisitorContext: &targeting.Context{
			Standard:             map[string]*structpb.Value{},
			IntegrationProviders: make(map[string]targeting.ContextMap),
		},
		Environment: &models.Environment{Common: &clientInfos},
	}

	// Check that at first view, only 1 ab test is returned
	err = Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))
	assert.Equal(t, 1, len(hitsProcessor.TrackedHits.CampaignActivations))
	assert.Equal(t, "vg1", hitsProcessor.TrackedHits.CampaignActivations[0].CampaignID)
	assert.Contains(t, hitsProcessor.TrackedHits.CampaignActivations[0].VariationID, "v_")
	assert.Equal(t, visID1, hitsProcessor.TrackedHits.CampaignActivations[0].VisitorID)
}

func TestDecision1Vis1Test(t *testing.T) {
	visID1 := "vis_id"
	clientID := "client_id"

	campaigns := []*common.Campaign{
		utils.CreateABCampaignMock(
			"campaign1",
			"vg1",
			utils.CreateAllUsersTargetingMock(),
			utils.CreateModification("key", "value", decision_response.ModificationsType_FLAG),
		),
		utils.CreateABCampaignMock(
			"campaign1bis",
			"vg1bis",
			utils.CreateAllUsersTargetingMock(),
			utils.CreateModification("key", "value", decision_response.ModificationsType_FLAG),
		),
	}
	clientInfos := common.Environment{
		ID:                clientID,
		Campaigns:         campaigns,
		IsPanic:           false,
		SingleAssignment:  true,
		UseReconciliation: false,
	}
	handleRequest := Request{
		DecisionContext: &connectors.DecisionContext{
			EnvID: clientID,
		},
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:  &wrapperspb.StringValue{Value: visID1},
			TriggerHit: &wrapperspb.BoolValue{Value: false},
			Context:    map[string]*structpb.Value{},
		},
		FullVisitorContext: &targeting.Context{
			Standard:             map[string]*structpb.Value{},
			IntegrationProviders: make(map[string]targeting.ContextMap),
		},
		Environment: &models.Environment{Common: &clientInfos},
	}

	// Check that at first view, only 1 ab test is returned
	err := Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))
}

func TestDecisionNoReconciliation(t *testing.T) {
	anonymousID := "1234"
	clientID := "client_id"
	handleRequest := Request{
		DecisionContext: &connectors.DecisionContext{
			EnvID: clientID,
		},
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:  &wrapperspb.StringValue{Value: anonymousID},
			TriggerHit: &wrapperspb.BoolValue{Value: false},
			Context:    map[string]*structpb.Value{},
		},
		FullVisitorContext: &targeting.Context{
			Standard:             map[string]*structpb.Value{},
			IntegrationProviders: make(map[string]targeting.ContextMap),
		},
	}

	campaigns := []*common.Campaign{
		utils.CreateABCampaignMock(
			"campaign2",
			"vg2",
			utils.CreateAllUsersTargetingMock(),
			utils.CreateModification("key", "value", decision_response.ModificationsType_FLAG),
		),
	}

	clientInfos := common.Environment{
		ID:                clientID,
		Campaigns:         campaigns,
		IsPanic:           false,
		SingleAssignment:  false,
		UseReconciliation: false,
	}
	handleRequest.Environment = &models.Environment{Common: &clientInfos}

	err := Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))

	assignVariationID := handleRequest.DecisionResponse.Campaigns[0].Variation.Id.Value
	variationIDs := []string{}
	for _, v := range campaigns[0].VariationGroups[0].Variations {
		variationIDs = append(variationIDs, v.ID)
	}
	assert.Contains(t, variationIDs, assignVariationID)

	// Login the user
	loggedInID := "vis_id_2"
	handleRequest.DecisionRequest.VisitorId = &wrapperspb.StringValue{Value: loggedInID}
	handleRequest.DecisionRequest.AnonymousId = &wrapperspb.StringValue{Value: anonymousID}

	err = Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))

	assert.NotEqual(t, assignVariationID, handleRequest.DecisionResponse.Campaigns[0].Variation.Id.Value)
}

func TestDecisionReconciliation(t *testing.T) {
	anonymousID := "1234"
	clientID := "client_id"
	handleRequest := Request{
		DecisionContext: utils.CreateMockDecisionContext(),
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:  &wrapperspb.StringValue{Value: anonymousID},
			TriggerHit: &wrapperspb.BoolValue{Value: false},
			Context:    map[string]*structpb.Value{},
		},
		FullVisitorContext: &targeting.Context{
			Standard:             map[string]*structpb.Value{},
			IntegrationProviders: make(map[string]targeting.ContextMap),
		},
	}

	campaigns := []*common.Campaign{
		utils.CreateABCampaignMock(
			"campaign3",
			"vg3",
			utils.CreateAllUsersTargetingMock(),
			utils.CreateModification("key", "value", decision_response.ModificationsType_FLAG),
		),
	}

	clientInfos := common.Environment{
		ID:                clientID,
		Campaigns:         campaigns,
		IsPanic:           false,
		SingleAssignment:  false,
		UseReconciliation: true,
		CacheEnabled:      true,
	}
	handleRequest.Environment = &models.Environment{Common: &clientInfos}

	err := Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))

	assignVariationID := handleRequest.DecisionResponse.Campaigns[0].Variation.Id.Value
	variationIDs := []string{}
	for _, v := range campaigns[0].VariationGroups[0].Variations {
		variationIDs = append(variationIDs, v.ID)
	}
	assert.Contains(t, variationIDs, assignVariationID)

	// Login the user
	loggedInID := "vis_id_logged"
	handleRequest.DecisionRequest.VisitorId = &wrapperspb.StringValue{Value: loggedInID}
	handleRequest.DecisionRequest.AnonymousId = &wrapperspb.StringValue{Value: anonymousID}

	err = Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))

	assert.Equal(t, assignVariationID, handleRequest.DecisionResponse.Campaigns[0].Variation.Id.Value)

	// Rehandle as if user is already logged in
	handleRequest.DecisionRequest.VisitorId = &wrapperspb.StringValue{Value: loggedInID}
	handleRequest.DecisionRequest.AnonymousId = nil

	err = Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))

	assert.Equal(t, assignVariationID, handleRequest.DecisionResponse.Campaigns[0].Variation.Id.Value)

	handleRequest.FullVisitorContext.IntegrationProviders["mixpanel"] = map[string]*structpb.Value{
		"age": structpb.NewStringValue("21"),
	}
	campaigns = []*common.Campaign{
		utils.CreateABCampaignMock(
			"campaign4",
			"vg3",
			utils.CreateTargetingWithProvider(),
			utils.CreateModification("key", "value", decision_response.ModificationsType_FLAG),
		),
	}
	clientInfos.Campaigns = campaigns

	err = Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))
}
