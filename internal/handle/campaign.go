package handle

import (
	common "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/decision_response"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func buildCampaignResponse(vg *common.VariationGroup, variation *common.Variation, shouldFillKeys bool) *decision_response.Campaign {
	campaignResponse := decision_response.Campaign{
		Id: &wrapperspb.StringValue{
			Value: vg.Campaign.ID,
		},
		VariationGroupId: &wrapperspb.StringValue{
			Value: vg.ID,
		},
	}

	if shouldFillKeys {
		if variation.Modifications == nil {
			variation.Modifications = &decision_response.Modifications{}
		}
		if variation.Modifications.Value == nil {
			variation.Modifications.Value = &structpb.Struct{}
		}
		if variation.Modifications.Value.Fields == nil {
			variation.Modifications.Value.Fields = map[string]*structpb.Value{}
		}
		for _, v := range vg.Variations {
			if v.Modifications != nil && v.Modifications.Value != nil && v.Modifications.Value.Fields != nil {
				for key := range v.Modifications.Value.Fields {
					if _, ok := variation.Modifications.Value.Fields[key]; !ok {
						variation.Modifications.Value.Fields[key] = &structpb.Value{Kind: &structpb.Value_NullValue{}}
					}
				}
			}
		}
	}

	protoModif := &decision_response.Variation{
		Id: &wrapperspb.StringValue{
			Value: variation.ID,
		},
		Modifications: variation.Modifications,
		Reference:     variation.Reference,
	}

	campaignResponse.Variation = protoModif
	return &campaignResponse
}
