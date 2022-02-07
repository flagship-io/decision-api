package experience_trackers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
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
	hits           []connectors.TrackingHit
	ticker         *time.Ticker
}

func NewDataCollectTracker() *DataCollectTracker {
	tracker := &DataCollectTracker{
		batchingWindow: defaultBatchingWindow,
		batchSize:      defaultBatchSize,
		hits:           []connectors.TrackingHit{},
		trackingURL:    defaultTrackingURL,
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
		sig := <-sigs
		fmt.Println(sig)
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
		log.Printf("error when marshaling batch hit: %v", err)
	}

	log.Printf("sending hits to datacollect: %v", string(json_data))
	req, err := http.NewRequest(http.MethodPost, d.trackingURL, bytes.NewBuffer(json_data))
	if err != nil {
		log.Printf("error when marshaling batch hit: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Printf("error when sending batch hit: %v", err)
		return
	}

	if resp.StatusCode >= 400 {
		log.Printf("error when sending batch hit: %v", resp.Status)
		return
	}

	d.hits = []connectors.TrackingHit{}
}

func (d *DataCollectTracker) TrackHits(hits []connectors.TrackingHit) error {
	d.hits = append(d.hits, hits...)
	if len(d.hits) >= d.batchSize {
		d.ticker.Reset(d.batchingWindow)
		d.sendBatchHit()
	}
	return nil
}
