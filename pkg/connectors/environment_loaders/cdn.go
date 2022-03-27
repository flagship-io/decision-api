package environment_loaders

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/bucketing"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

const defaultBaseURL = "https://cdn-staging.flagship.io"
const defaultTimeout = time.Second * 5
const defaultPollingInterval = time.Second * 5
const logName = "CDN Loader"

type CDNLoader struct {
	baseURL           string
	httpClient        *http.Client
	lastModified      string
	timeout           time.Duration
	pollingInternal   time.Duration
	loadedEnvironment *models.Environment
	logger            *logger.Logger
	lock              *sync.RWMutex
}

type CDNLoaderOptionBuilder func(*CDNLoader)

func WithBaseURL(url string) CDNLoaderOptionBuilder {
	return func(l *CDNLoader) {
		l.baseURL = url
	}
}

func WithPollingInterval(pollingInterval time.Duration) CDNLoaderOptionBuilder {
	return func(l *CDNLoader) {
		l.pollingInternal = pollingInterval
	}
}

func WithLogLevel(lvl string) CDNLoaderOptionBuilder {
	return func(l *CDNLoader) {
		l.logger = logger.New(lvl, logName)
	}
}

func WithHTTPClient(client *http.Client) CDNLoaderOptionBuilder {
	return func(l *CDNLoader) {
		l.httpClient = client
	}
}

func NewCDNLoader(opts ...CDNLoaderOptionBuilder) *CDNLoader {
	loader := &CDNLoader{
		baseURL:         defaultBaseURL,
		httpClient:      &http.Client{},
		timeout:         defaultTimeout,
		pollingInternal: defaultPollingInterval,
		logger:          logger.New(logrus.WarnLevel.String(), logName),
		lock:            &sync.RWMutex{},
	}

	for _, o := range opts {
		o(loader)
	}

	loader.httpClient.Timeout = loader.timeout

	return loader
}

func (loader *CDNLoader) Init(envID string, APIKey string) error {
	ticker := time.NewTicker(loader.pollingInternal)
	done := make(chan bool, 1)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	loader.logger.Info("initializing CDN environment loader")

	go func() {
		for {
			select {
			case <-done:
				os.Exit(0)
			case <-ticker.C:
				err := loader.fetchEnvironment(envID, APIKey)
				if err != nil {
					loader.logger.Errorf("error when fetching environment: %v", err)
				}
			}
		}
	}()

	go func() {
		// When receiving sigterm signal, send an event to the done channel
		<-sigs
		done <- true
	}()

	return loader.fetchEnvironment(envID, APIKey)
}

func (l *CDNLoader) fetchEnvironment(envID string, APIKey string) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s/bucketing.json", l.baseURL, envID), nil)
	if err != nil {
		return fmt.Errorf("an error occured when creating HTTP request: %v", err)
	}

	l.lock.RLock()
	req.Header.Set("If-Modified-Since", l.lastModified)
	l.lock.RUnlock()

	resp, err := l.httpClient.Do(req)
	if err != nil {
		l.logger.Errorf("an error occured when fetching environment: %v", err)
		return err
	}

	if resp.StatusCode >= 400 {
		l.logger.Errorf("an HTTP error occured when fetching environment: %v", resp.Status)
		return errors.New("environment loader HTTP error")
	}

	if resp.StatusCode == 304 {
		return nil
	}

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.logger.Errorf("error when reading body: %v", err)
		return err
	}

	conf := &bucketing.Bucketing_BucketingResponse{}
	err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(response, conf)

	if err != nil {
		l.logger.Errorf("an error occured when parsing environment: %v", err)
		return err
	}

	campaigns := []*common.Campaign{}
	for _, c := range conf.Campaigns {
		campaigns = append(campaigns, campaignToCommonStruct(c))
	}

	l.lock.Lock()
	l.loadedEnvironment = &models.Environment{
		Common: &common.Environment{
			ID:                envID,
			Campaigns:         campaigns,
			IsPanic:           conf.Panic,
			SingleAssignment:  conf.AccountSettings.Enabled1V1T,
			UseReconciliation: conf.AccountSettings.EnabledXPC || conf.VisitorConsolidation,
			CacheEnabled:      true,
		},
		HasIntegrations: false,
	}
	l.lastModified = resp.Header.Get("Last-Modified")
	l.lock.Unlock()
	l.logger.Infof("environment with id %s loaded", envID)

	return nil
}

func variationToCommonStruct(v *decision_response.FullVariation) *common.Variation {
	return &common.Variation{
		ID:            v.Id.Value,
		Reference:     v.Reference,
		Allocation:    float32(v.Allocation),
		Modifications: v.Modifications,
	}
}

func variationGroupToCommonStruct(vg *bucketing.Bucketing_BucketingVariationGroups, campaign *bucketing.Bucketing_BucketingCampaign) *common.VariationGroup {
	variations := []*common.Variation{}
	for _, v := range vg.Variations {
		variations = append(variations, variationToCommonStruct(v))
	}
	bucketRange := [][]float64{}
	for _, r := range campaign.BucketRanges {
		bucketRange = append(bucketRange, r.R)
	}
	return &common.VariationGroup{
		ID: vg.Id,
		Campaign: &common.Campaign{
			ID:           campaign.Id,
			Type:         campaign.Type,
			BucketRanges: bucketRange,
		},
		Targetings: vg.Targeting,
		Variations: variations,
	}
}

func campaignToCommonStruct(c *bucketing.Bucketing_BucketingCampaign) *common.Campaign {
	variationGroups := []*common.VariationGroup{}
	for _, vg := range c.VariationGroups {
		variationGroups = append(variationGroups, variationGroupToCommonStruct(vg, c))
	}
	bucketRange := [][]float64{}
	for _, r := range c.BucketRanges {
		bucketRange = append(bucketRange, r.R)
	}
	var slug *string = nil
	if c.Slug != nil {
		slug = &(c.Slug.Value)
	}
	return &common.Campaign{
		ID:              c.Id,
		Slug:            slug,
		Type:            c.Type,
		VariationGroups: variationGroups,
		BucketRanges:    bucketRange,
	}
}

func (l *CDNLoader) LoadEnvironment(envID string, APIKey string) (*models.Environment, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var err error
	if l.loadedEnvironment == nil {
		err = l.fetchEnvironment(envID, APIKey)
	}

	// create a new environment to prevent reference modifications
	environment := &models.Environment{
		Common: &common.Environment{
			ID:                envID,
			Campaigns:         make([]*common.Campaign, len(l.loadedEnvironment.Common.Campaigns)),
			IsPanic:           l.loadedEnvironment.Common.IsPanic,
			SingleAssignment:  l.loadedEnvironment.Common.SingleAssignment,
			UseReconciliation: l.loadedEnvironment.Common.UseReconciliation,
			CacheEnabled:      l.loadedEnvironment.Common.CacheEnabled,
		},
		HasIntegrations: l.loadedEnvironment.HasIntegrations,
	}
	copy(environment.Common.Campaigns, l.loadedEnvironment.Common.Campaigns)
	return environment, err
}
