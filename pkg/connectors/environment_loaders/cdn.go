package environment_loaders

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
	"github.com/flagship-io/flagship-proto/bucketing"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

const defaultBaseURL = "https://cdn.flagship.io"
const defaultTimeout = time.Second * 2
const defaultPollingInterval = time.Second * 5
const logName = "CDN Loader"

type CDNLoader struct {
	baseURL           string
	lastModified      string
	timeout           time.Duration
	pollingInternal   time.Duration
	loadedEnvironment *common.Environment
	logger            *logger.Logger
}

type CDNLoaderOptionBuilder func(*CDNLoader)

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

func NewCDNLoader(opts ...CDNLoaderOptionBuilder) *CDNLoader {
	loader := &CDNLoader{
		baseURL:         defaultBaseURL,
		timeout:         defaultTimeout,
		pollingInternal: defaultPollingInterval,
		logger:          logger.New(logrus.WarnLevel.String(), logName),
	}

	for _, o := range opts {
		o(loader)
	}

	return loader
}

func (loader *CDNLoader) Init(envID string, APIKey string) error {
	ticker := time.NewTicker(loader.pollingInternal)
	done := make(chan bool, 1)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	loader.logger.Info("initialize CDN loader")

	go func() {
		for {
			select {
			case <-done:
				os.Exit(0)
			case <-ticker.C:
				loader.fetchEnvironment(envID, APIKey)
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
	client := &http.Client{
		Timeout: l.timeout,
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s/bucketing.json", l.baseURL, envID), nil)
	if err != nil {
		return fmt.Errorf("an error occured when creating HTTP request: %v", err)
	}

	req.Header.Set("If-Modified-Since", l.lastModified)
	resp, err := client.Do(req)
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

	l.loadedEnvironment = &common.Environment{
		ID:                envID,
		Campaigns:         campaigns,
		IsPanic:           conf.Panic,
		SingleAssignment:  conf.AccountSettings.Enabled1V1T,
		UseReconciliation: conf.VisitorConsolidation,
		CacheEnabled:      true,
	}
	l.lastModified = resp.Header.Get("Last-Modified")
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

func (l *CDNLoader) LoadEnvironment(envID string, APIKey string) (*common.Environment, error) {
	if l.loadedEnvironment == nil {
		err := l.fetchEnvironment(envID, APIKey)
		return l.loadedEnvironment, err
	}
	return l.loadedEnvironment, nil
}
