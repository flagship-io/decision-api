package hits_processors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

// defaultBatchingWindow is the default time duration for batching hits.
const defaultBatchingWindow = time.Second * 30

// defaultBatchSize is the default number of hits to include in a batch.
const defaultBatchSize = 50

// defaultTrackingURL is the default URL to send batched hits to.
const defaultTrackingURL = "https://ariane.abtasty.com"

// defaultLogLevel is the default log level for the DataCollect Processor.
const defaultLogLevel = "error"

// logName is the name of the logger used by the DataCollect Processor.
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
	ticker         chan time.Time
	lastTick       time.Time
	logger         *logger.Logger
	httpClient     *http.Client
	lock           *sync.Mutex
}

type DatacollectOptionBuilder func(*DataCollectProcessor)

// WithBatchOptions is an option function that sets the batch size and window for the DataCollectProcessor.
func WithBatchOptions(batchSize int, batchingWindow time.Duration) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.batchSize = batchSize
		l.batchingWindow = batchingWindow
	}
}

// WithTrackingURL is an option function that sets the tracking URL for the DataCollectProcessor.
func WithTrackingURL(url string) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.trackingURL = url
	}
}

// WithLogger is an option function that sets the logger for the DataCollectProcessor.
func WithLogger(lvl string, fmt logger.LogFormat) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.logger = logger.New(lvl, fmt, logName)
	}
}

// WithHTTPClient is an option function that sets the HTTP client for the DataCollectProcessor.
func WithHTTPClient(client *http.Client) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.httpClient = client
	}
}

// NewDataCollectProcessor creates a new DataCollectProcessor with the given options.
func NewDataCollectProcessor(opts ...DatacollectOptionBuilder) *DataCollectProcessor {
	processor := &DataCollectProcessor{
		batchingWindow: defaultBatchingWindow,
		batchSize:      defaultBatchSize,
		hits:           []models.MappableHit{},
		trackingURL:    defaultTrackingURL,
		logger:         logger.New(defaultLogLevel, logger.FORMAT_TEXT, logName),
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		lock: &sync.Mutex{},
	}

	for _, o := range opts {
		o(processor)
	}

	processor.logger.Info("initializing datacollect hits processor")
	processor.ticker = make(chan time.Time)

	go func() {
		for {
			time.Sleep(processor.batchingWindow)
			processor.lock.Lock()
			durationSinceLastTick := time.Since(processor.lastTick)
			// If last tick was trigger in between because of full batch, wait a little more
			if durationSinceLastTick < processor.batchingWindow {
				time.Sleep(processor.batchingWindow - durationSinceLastTick)
			}
			processor.lock.Unlock()
			processor.ticker <- time.Now()
		}
	}()

	go func() {
		for t := range processor.ticker {
			processor.sendHits(processor.hits, t)
			processor.hits = []models.MappableHit{}
		}
	}()

	return processor
}

// sendBatchHit sends a batch of hits to the trackingURL using the httpClient.
func (d *DataCollectProcessor) sendBatchHit(ctx context.Context, mappableHits []models.MappableHit) error {
	if len(mappableHits) == 0 {
		d.logger.Info("no hits to send")
		return nil
	}

	hits := []map[string]interface{}{}
	for _, h := range mappableHits {
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

	d.logger.Infof("sending %d hits to datacollect: %v", len(batchHit.Hits), string(json_data))
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

	return nil
}

func (d *DataCollectProcessor) sendHits(hits []models.MappableHit, tick time.Time) {
	err := d.sendBatchHit(context.Background(), hits)
	if err != nil {
		d.logger.Errorf("error when sending batch hit: %v", err)
	}
	d.lock.Lock()
	d.lastTick = tick
	d.lock.Unlock()
}

// TrackHits adds the given hits to the processor for tracking.
// If the number of hits in the processor exceeds the batch size, a batch of hits is sent.
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
		go d.sendHits(d.hits, time.Now())
		d.lock.Lock()
		d.hits = []models.MappableHit{}
		d.lock.Unlock()
	}
	return nil
}

func (d *DataCollectProcessor) Shutdown(ctx context.Context) error {
	return d.sendBatchHit(ctx, d.hits)
}
