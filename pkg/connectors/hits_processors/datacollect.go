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

type batchHit struct {
	Type       string                   `json:"t"`
	DataSource string                   `json:"ds"`
	Hits       []map[string]interface{} `json:"h"`
}

type DataCollectTracker struct {
	batchingWindow time.Duration
	batchSize      int
	trackingURL    string
	hits           []models.MappableHit
	ticker         *time.Ticker
	logger         *logger.Logger
}

func NewDataCollectTracker(logLevel string) *DataCollectTracker {
	tracker := &DataCollectTracker{
		batchingWindow: defaultBatchingWindow,
		batchSize:      defaultBatchSize,
		hits:           []models.MappableHit{},
		trackingURL:    defaultTrackingURL,
		logger:         logger.New(logLevel, "DataCollect Tracker"),
	}

	tracker.ticker = time.NewTicker(tracker.batchingWindow)
	done := make(chan bool, 1)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-done:
				tracker.sendBatchHit()
				os.Exit(0)
			case <-tracker.ticker.C:
				tracker.sendBatchHit()
			}
		}
	}()

	go func() {
		// When receiving sigterm signal, send an event to the done channel
		<-sigs
		done <- true
	}()

	return tracker
}

func (d *DataCollectTracker) sendBatchHit() {
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

	resp, err := http.DefaultClient.Do(req)

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

func (d *DataCollectTracker) TrackHits(hits connectors.TrackingHits) error {
	mappableHits := []models.MappableHit{}
	for _, ca := range hits.CampaignActivations {
		mappableHits = append(mappableHits, ca)
	}
	d.hits = append(d.hits, mappableHits...)
	if len(d.hits) >= d.batchSize {
		d.ticker.Reset(d.batchingWindow)
		d.sendBatchHit()
	}
	return nil
}
