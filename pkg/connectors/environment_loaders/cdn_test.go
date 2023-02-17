package environment_loaders

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/logger"
	"github.com/flagship-io/flagship-proto/bucketing"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/flagship-io/flagship-proto/targeting"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestNewCDNLoader(t *testing.T) {
	lock := &sync.Mutex{}
	conf := &bucketing.Bucketing_BucketingResponse{
		Panic: true,
		Campaigns: []*bucketing.Bucketing_BucketingCampaign{
			{
				Id:   "cid",
				Slug: wrapperspb.String("slug"),
				Type: "type",
				VariationGroups: []*bucketing.Bucketing_BucketingVariationGroups{
					{
						Id:        "vgid",
						Targeting: &targeting.Targeting{},
						Variations: []*decision_response.FullVariation{
							{
								Id:         wrapperspb.String("vid"),
								Reference:  false,
								Allocation: 100,
								Modifications: &decision_response.Modifications{
									Type: decision_response.ModificationsType_FLAG,
									Value: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"flag": structpb.NewBoolValue(true),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		AccountSettings: &decision_response.AccountSettings{
			EnabledXPC:  true,
			Enabled1V1T: true,
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		lock.Lock()
		confJSON, _ := protojson.Marshal(conf)
		_, err := rw.Write(confJSON)
		assert.Nil(t, err)
		lock.Unlock()
	}))
	// Close the server when test finishes
	defer server.Close()

	httpClient := &http.Client{}
	loader := NewCDNLoader(WithBaseURL(server.URL), WithLogger("debug", logger.FORMAT_TEXT), WithPollingInterval(time.Second*1), WithHTTPClient(httpClient))

	assert.NotNil(t, loader)
	assert.Equal(t, server.URL, loader.baseURL)
	assert.Equal(t, time.Second*1, loader.pollingInternal)
	assert.Equal(t, httpClient, loader.httpClient)

	err := loader.Init("env_id", "api_key")
	assert.Nil(t, err)
	campaign := loader.loadedEnvironment.Common.Campaigns[0]
	assert.EqualValues(t, conf.Panic, loader.loadedEnvironment.Common.IsPanic)
	assert.EqualValues(t, conf.AccountSettings.Enabled1V1T, loader.loadedEnvironment.Common.SingleAssignment)
	assert.EqualValues(t, conf.AccountSettings.EnabledXPC, loader.loadedEnvironment.Common.UseReconciliation)
	assert.EqualValues(t, conf.Campaigns[0].Id, campaign.ID)
	assert.EqualValues(t, conf.Campaigns[0].Slug.Value, *campaign.Slug)
	assert.EqualValues(t, conf.Campaigns[0].Type, campaign.Type)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Id, campaign.VariationGroups[0].ID)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Targeting, campaign.VariationGroups[0].Targetings)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Id.Value, campaign.VariationGroups[0].Variations[0].ID)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Allocation, campaign.VariationGroups[0].Variations[0].Allocation)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Reference, campaign.VariationGroups[0].Variations[0].Reference)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Modifications, campaign.VariationGroups[0].Variations[0].Modifications)

	_, err = loader.LoadEnvironment("env_id", "api_key")
	assert.Nil(t, err)

	lock.Lock()
	conf.Panic = false
	lock.Unlock()
	time.Sleep(time.Second * 1)

	lock.Lock()
	data, err := loader.LoadEnvironment("env_id", "api_key")
	assert.Nil(t, err)
	assert.Equal(t, conf.Panic, data.Common.IsPanic)
	lock.Unlock()
}
