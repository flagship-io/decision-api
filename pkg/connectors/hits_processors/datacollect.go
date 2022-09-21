package hits_processors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

const defaultBatchingWindow = time.Second * 30
const defaultBatchSize = 50
const defaultTrackingURL = "https://events.flagship.io"
const defaultLogLevel = "error"
const logName = "DataCollect Processor"

type batchHit struct {
	Type            string                   `json:"t"`
	DataSource      string                   `json:"ds"`
	Hits            []map[string]interface{} `json:"h"`
	CustomVariables map[string]string        `json:"cv"`
}

type DataCollectProcessor struct {
	batchingWindow time.Duration
	batchSize      int
	trackingURL    string
	hits           []models.MappableHit
	ticker         *time.Ticker
	logger         *logger.Logger
	httpClient     *http.Client
}

type DatacollectOptionBuilder func(*DataCollectProcessor)

func WithBatchOptions(batchSize int, batchingWindow time.Duration) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.batchSize = batchSize
		l.batchingWindow = batchingWindow
	}
}

func WithTrackingURL(url string) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.trackingURL = url
	}
}

func WithLogLevel(lvl string) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.logger = logger.New(lvl, logName)
	}
}

func WithHTTPClient(client *http.Client) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.httpClient = client
	}
}

func NewDataCollectProcessor(opts ...DatacollectOptionBuilder) *DataCollectProcessor {
	processor := &DataCollectProcessor{
		batchingWindow: defaultBatchingWindow,
		batchSize:      defaultBatchSize,
		hits:           []models.MappableHit{},
		trackingURL:    defaultTrackingURL,
		logger:         logger.New(defaultLogLevel, logName),
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}

	for _, o := range opts {
		o(processor)
	}

	processor.logger.Info("initializing datacollect hits processor")
	processor.ticker = time.NewTicker(processor.batchingWindow)

	go func() {
		for range processor.ticker.C {
			err := processor.sendBatchHit(context.Background())
			if err != nil {
				processor.logger.Errorf("error when sending batch hit: %v", err)
			}
		}
	}()

	return processor
}

func (d *DataCollectProcessor) sendBatchHit(ctx context.Context) error {
	if len(d.hits) == 0 {
		d.logger.Info("no hits to send")
		return nil
	}

	hits := []map[string]interface{}{}
	for _, h := range d.hits {
		h.ComputeQueueTime()
		hits = append(hits, h.ToMap())
	}

	batchHit := &batchHit{
		Type:       "BATCH",
		DataSource: "APP",
		Hits:       hits,
		CustomVariables: map[string]string{
			"0": "runner, self-hosted",
			"1": fmt.Sprintf("version, %s", models.Version),
			"2": fmt.Sprintf("go-version, %s", runtime.Version()),
		},
	}

	json_data, err := json.Marshal(batchHit)
	if err != nil {
		return fmt.Errorf("error when marshaling batch hit: %v", err)
	}

	d.logger.Infof("sending hits to datacollect: %v", string(json_data))
	req, err := http.NewRequest(http.MethodPost, d.trackingURL, bytes.NewBuffer(json_data))
	req = req.WithContext(ctx)
	if err != nil {
		d.logger.Errorf("error when marshaling batch hit: %v", err)
	}

	resp, err := d.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("error when making HTTP request: %v", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("got status %v when calling HTTP request", resp.Status)
	}
	d.logger.Infof("%d hits sent to datacollect successfully", len(hits))

	d.hits = []models.MappableHit{}
	return nil
}

func (d *DataCollectProcessor) TrackHits(hits connectors.TrackingHits) error {
	mappableHits := []models.MappableHit{}
	for _, ca := range hits.CampaignActivations {
		mappableHits = append(mappableHits, ca)
	}
	for _, vc := range hits.VisitorContext {
		mappableHits = append(mappableHits, vc)
	}
	d.hits = append(d.hits, mappableHits...)
	if len(d.hits) >= d.batchSize {
		d.ticker.Reset(d.batchingWindow)
		err := d.sendBatchHit(context.Background())
		if err != nil {
			d.logger.Errorf("error when sending batch hit: %v", err)
		}
	}
	return nil
}

func (d *DataCollectProcessor) Shutdown(ctx context.Context) error {
	return d.sendBatchHit(ctx)
}
