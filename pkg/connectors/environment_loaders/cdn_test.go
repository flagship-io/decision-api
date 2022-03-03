package environment_loaders

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/flagship-io/flagship-proto/bucketing"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/flagship-io/flagship-proto/targeting"
	"github.com/sirupsen/logrus"
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
		rw.Write(confJSON)
		lock.Unlock()
	}))
	// Close the server when test finishes
	defer server.Close()

	httpClient := &http.Client{}
	loader := NewCDNLoader(WithBaseURL(server.URL), WithLogLevel(logrus.DebugLevel.String()), WithPollingInterval(time.Second*1), WithHTTPClient(httpClient))

	assert.NotNil(t, loader)
	assert.Equal(t, server.URL, loader.baseURL)
	assert.Equal(t, time.Second*1, loader.pollingInternal)
	assert.Equal(t, httpClient, loader.httpClient)

	loader.Init("env_id", "api_key")
	assert.EqualValues(t, conf.Panic, loader.loadedEnvironment.IsPanic)
	assert.EqualValues(t, conf.AccountSettings.Enabled1V1T, loader.loadedEnvironment.SingleAssignment)
	assert.EqualValues(t, conf.AccountSettings.EnabledXPC, loader.loadedEnvironment.UseReconciliation)
	assert.EqualValues(t, conf.Campaigns[0].Id, loader.loadedEnvironment.Campaigns[0].ID)
	assert.EqualValues(t, conf.Campaigns[0].Slug.Value, *loader.loadedEnvironment.Campaigns[0].Slug)
	assert.EqualValues(t, conf.Campaigns[0].Type, loader.loadedEnvironment.Campaigns[0].Type)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Id, loader.loadedEnvironment.Campaigns[0].VariationGroups[0].ID)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Targeting, loader.loadedEnvironment.Campaigns[0].VariationGroups[0].Targetings)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Id.Value, loader.loadedEnvironment.Campaigns[0].VariationGroups[0].Variations[0].ID)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Allocation, loader.loadedEnvironment.Campaigns[0].VariationGroups[0].Variations[0].Allocation)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Reference, loader.loadedEnvironment.Campaigns[0].VariationGroups[0].Variations[0].Reference)
	assert.EqualValues(t, conf.Campaigns[0].VariationGroups[0].Variations[0].Modifications, loader.loadedEnvironment.Campaigns[0].VariationGroups[0].Variations[0].Modifications)

	_, err := loader.LoadEnvironment("env_id", "api_key")
	assert.Nil(t, err)

	lock.Lock()
	conf.Panic = false
	lock.Unlock()
	time.Sleep(time.Second * 1)

	lock.Lock()
	data, err := loader.LoadEnvironment("env_id", "api_key")
	assert.Nil(t, err)
	assert.EqualValues(t, conf.Panic, data.IsPanic)
	lock.Unlock()
}
