package handle

import (
	"testing"

	"github.com/flagship-io/decision-api/internal/utils"

	"gitlab.com/canarybay/protobuf/ptypes.git/decision_request"

	common "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/decision_response"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"

	structpb "github.com/golang/protobuf/ptypes/struct"
)

var vgTest *common.VariationGroup = &common.VariationGroup{
	Campaign: &common.Campaign{
		ID: "testCampaignId",
	},
	ID: "testId",
	Variations: []*common.Variation{
		&common.Variation{
			Modifications: &decision_response.Modifications{
				Value: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"test": &structpb.Value{
							Kind: &structpb.Value_StringValue{
								StringValue: "youpi",
							},
						},
					},
				},
			},
			Allocation: 50,
		}, &common.Variation{
			Modifications: &decision_response.Modifications{
				Value: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"test2": &structpb.Value{
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
		DecisionContext: &DecisionContext{
			EnvID: clientID,
		},
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:  &wrappers.StringValue{Value: visID1},
			TriggerHit: &wrappers.BoolValue{Value: false},
			Context:    map[string]*structpb.Value{},
		},
		Environment: &clientInfos,
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
		DecisionContext: &DecisionContext{
			EnvID: clientID,
		},
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:  &wrappers.StringValue{Value: anonymousID},
			TriggerHit: &wrappers.BoolValue{Value: false},
			Context:    map[string]*structpb.Value{},
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
	handleRequest.Environment = &clientInfos

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
	handleRequest.DecisionRequest.VisitorId = &wrappers.StringValue{Value: loggedInID}
	handleRequest.DecisionRequest.AnonymousId = &wrappers.StringValue{Value: anonymousID}

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
		DecisionContext: &DecisionContext{
			EnvID: clientID,
		},
		DecisionRequest: &decision_request.DecisionRequest{
			VisitorId:  &wrappers.StringValue{Value: anonymousID},
			TriggerHit: &wrappers.BoolValue{Value: false},
			Context:    map[string]*structpb.Value{},
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
	handleRequest.Environment = &clientInfos

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
	handleRequest.DecisionRequest.VisitorId = &wrappers.StringValue{Value: loggedInID}
	handleRequest.DecisionRequest.AnonymousId = &wrappers.StringValue{Value: anonymousID}

	err = Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))

	assert.Equal(t, assignVariationID, handleRequest.DecisionResponse.Campaigns[0].Variation.Id.Value)

	// Rehandle as if user is already logged in
	handleRequest.DecisionRequest.VisitorId = &wrappers.StringValue{Value: loggedInID}
	handleRequest.DecisionRequest.AnonymousId = nil

	err = Decision(&handleRequest, nil)
	assert.Nil(t, err)
	assert.NotNil(t, handleRequest.DecisionResponse)
	assert.Equal(t, 1, len(handleRequest.DecisionResponse.Campaigns))

	assert.Equal(t, assignVariationID, handleRequest.DecisionResponse.Campaigns[0].Variation.Id.Value)
}
