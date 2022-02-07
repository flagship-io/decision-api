package utils

import (
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
	}
}
