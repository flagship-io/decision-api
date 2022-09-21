package utils

import (
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/flagship-io/flagship-proto/targeting"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CreateAllUsersTargetingMock creates a mock targeting all users as byte
func CreateAllUsersTargetingMock() *targeting.Targeting {
	return &targeting.Targeting{
		TargetingGroups: []*targeting.Targeting_TargetingGroup{
			{
				Targetings: []*targeting.Targeting_InnerTargeting{
					{
						Operator: targeting.Targeting_EQUALS,
						Key:      &wrapperspb.StringValue{Value: "fs_all_users"},
					},
				},
			},
		},
	}
}

// CreateTargetingWithProvider creates a mock targeting with an integration provider
func CreateTargetingWithProvider() *targeting.Targeting {
	return &targeting.Targeting{
		TargetingGroups: []*targeting.Targeting_TargetingGroup{
			{
				Targetings: []*targeting.Targeting_InnerTargeting{
					{
						Operator: targeting.Targeting_GREATER_THAN,
						Key:      &wrapperspb.StringValue{Value: "age"},
						Value:    structpb.NewStringValue("20"),
						Provider: &wrapperspb.StringValue{Value: "mixpanel"},
					},
				},
			},
		},
	}
}

// CreateModification returns a single modification with key value string as byte array
func CreateModification(key string, value interface{}, modifType decision_response.ModificationsType) *decision_response.Modifications {
	// Modif of type flag string
	modifValue := &structpb.Struct{}
	modifValue.Fields = make(map[string]*structpb.Value)
	modifValue.Fields[key], _ = structpb.NewValue(value)
	return &decision_response.Modifications{
		Type:  modifType,
		Value: modifValue,
	}
}

// CreateABCampaignMock returns a mocked AB Test campaign
func CreateABCampaignMock(campaignID string, vgID string, targetings *targeting.Targeting, modifications *decision_response.Modifications) *common.Campaign {
	variations := []*common.Variation{
		{
			Allocation:    50,
			ID:            "v_1",
			Modifications: modifications,
		},
		{
			Allocation:    50,
			ID:            "v_2",
			Modifications: modifications,
		},
	}

	lastAllocation := float32(0.)
	variationsArray := []*common.Variation{}
	for _, v := range variations {
		lastAllocation = v.Allocation + lastAllocation
		newV := &common.Variation{
			Allocation:    lastAllocation,
			ID:            v.ID,
			Modifications: modifications,
		}
		variationsArray = append(variationsArray, newV)
	}

	return &common.Campaign{
		ID:   campaignID,
		Type: "ab",
		VariationGroups: []*common.VariationGroup{
			{
				ID: vgID,
				Campaign: &common.Campaign{
					ID:   campaignID,
					Type: "ab",
				},
				Targetings: targetings,
				Variations: variationsArray,
			},
		},
		BucketRanges: [][]float64{{0, 100}},
	}
}

func CreateMockDecisionContext() *connectors.DecisionContext {
	modifications := CreateModification("testString", "string", decision_response.ModificationsType_FLAG)
	modifications.Value.Fields["testBool"], _ = structpb.NewValue(true)
	modifications.Value.Fields["testNumber"], _ = structpb.NewValue(11.)
	modifications.Value.Fields["testWhatever"], _ = structpb.NewValue([]interface{}{"a", 1.})

	return &connectors.DecisionContext{
		EnvID:  "env_id_1",
		APIKey: "api_key_id",
		Logger: logger.New("debug", logger.FORMAT_TEXT, "mock"),
		Connectors: connectors.Connectors{
			HitsProcessor:      &hits_processors.MockHitProcessor{},
			AssignmentsManager: assignments_managers.InitMemoryManager(),
			EnvironmentLoader: &environment_loaders.MockLoader{
				MockedEnvironment: &models.Environment{
					Common: &common.Environment{
						Campaigns: []*common.Campaign{
							CreateABCampaignMock(
								"campaign_1",
								"vg_2",
								CreateAllUsersTargetingMock(),
								modifications),
							CreateABCampaignMock(
								"image",
								"vg_1",
								CreateAllUsersTargetingMock(),
								CreateModification("image", "http://image.jpeg", decision_response.ModificationsType_IMAGE)),
							CreateABCampaignMock(
								"campaign_2",
								"vg_3",
								CreateTargetingWithProvider(),
								modifications),
						},
					},
					HasIntegrations: true,
				},
			},
		},
	}
}
