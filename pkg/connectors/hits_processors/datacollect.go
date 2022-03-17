package hits_processors

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

const defaultBatchingWindow = time.Second * 30
const defaultBatchSize = 50
const defaultTrackingURL = "https://ariane.abtasty.com"
const defaultLogLevel = "error"
const logName = "DataCollect Processor"

type batchHit struct {
	Type       string                   `json:"t"`
	DataSource string                   `json:"ds"`
	Hits       []map[string]interface{} `json:"h"`
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

	processor.ticker = time.NewTicker(processor.batchingWindow)
	done := make(chan bool, 1)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-done:
				processor.sendBatchHit()
				os.Exit(0)
			case <-processor.ticker.C:
				processor.sendBatchHit()
			}
		}
	}()

	go func() {
		// When receiving sigterm signal, send an event to the done channel
		<-sigs
		done <- true
	}()

	return processor
}

func (d *DataCollectProcessor) sendBatchHit() {
	if len(d.hits) == 0 {
		return
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
	}

	json_data, err := json.Marshal(batchHit)
	if err != nil {
		d.logger.Errorf("error when marshaling batch hit: %v", err)
	}

	d.logger.Infof("sending hits to datacollect: %v", string(json_data))
	req, err := http.NewRequest(http.MethodPost, d.trackingURL, bytes.NewBuffer(json_data))
	if err != nil {
		d.logger.Errorf("error when marshaling batch hit: %v", err)
	}

	resp, err := d.httpClient.Do(req)

	if err != nil {
		d.logger.Errorf("error when sending batch hit: %v", err)
		return
	}

	if resp.StatusCode >= 400 {
		d.logger.Errorf("error when sending batch hit: %v", resp.Status)
		return
	}
	d.logger.Infof("%d hits sent to datacollect successfully", len(hits))

	d.hits = []models.MappableHit{}
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
		d.sendBatchHit()
	}
	return nil
}
