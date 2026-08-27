package hits_processors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// defaultMaxBatchBytes caps the size of a single request body. A batch larger than this is split
// into several requests, so a slow link cannot leave the collector with a half-written body.
const defaultMaxBatchBytes = 512 * 1024

// defaultSendRetries is how many extra attempts a failed request gets.
const defaultSendRetries = 2

// defaultRetryDelay is the delay before the first retry. It grows with each attempt.
const defaultRetryDelay = 200 * time.Millisecond

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
	maxBatchBytes  int
	sendRetries    int
	retryDelay     time.Duration
	trackingURL    string
	hits           []models.MappableHit
	ticker         chan time.Time
	lastTick       time.Time
	logger         *logger.Logger
	httpClient     *http.Client
	lock           *sync.Mutex
	stop           chan struct{}
	stopOnce       sync.Once
	running        sync.WaitGroup
}

type DatacollectOptionBuilder func(*DataCollectProcessor)

// WithBatchOptions is an option function that sets the batch size and window for the DataCollectProcessor.
func WithBatchOptions(batchSize int, batchingWindow time.Duration) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		l.batchSize = batchSize
		l.batchingWindow = batchingWindow
	}
}

// WithSendOptions is an option function that sets the maximum request body size, how many extra
// attempts a failed request gets, and the delay before the first retry. Values that would stop hits
// from being sent at all fall back to the defaults.
func WithSendOptions(maxBatchBytes int, sendRetries int, retryDelay time.Duration) DatacollectOptionBuilder {
	return func(l *DataCollectProcessor) {
		if maxBatchBytes <= 0 {
			maxBatchBytes = defaultMaxBatchBytes
		}
		if sendRetries < 0 {
			sendRetries = 0
		}
		if retryDelay < 0 {
			retryDelay = defaultRetryDelay
		}
		l.maxBatchBytes = maxBatchBytes
		l.sendRetries = sendRetries
		l.retryDelay = retryDelay
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
		maxBatchBytes:  defaultMaxBatchBytes,
		sendRetries:    defaultSendRetries,
		retryDelay:     defaultRetryDelay,
		hits:           []models.MappableHit{},
		trackingURL:    defaultTrackingURL,
		logger:         logger.New(defaultLogLevel, logger.FORMAT_TEXT, logName),
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		lock: &sync.Mutex{},
		stop: make(chan struct{}),
	}

	for _, o := range opts {
		o(processor)
	}

	processor.logger.Info("initializing datacollect hits processor")
	processor.ticker = make(chan time.Time)

	// Both goroutines are counted before they start, so Shutdown always waits for a flush that is
	// already running - including one started by the batching window rather than by a full batch.
	processor.running.Add(2)

	go func() {
		defer processor.running.Done()
		for {
			if !processor.wait(processor.batchingWindow) {
				return
			}
			processor.lock.Lock()
			durationSinceLastTick := time.Since(processor.lastTick)
			processor.lock.Unlock()
			// If last tick was triggered in between because of full batch, wait a little more.
			// Never hold the lock while waiting: TrackHits takes it on every request.
			if durationSinceLastTick < processor.batchingWindow {
				if !processor.wait(processor.batchingWindow - durationSinceLastTick) {
					return
				}
			}
			select {
			case processor.ticker <- time.Now():
			case <-processor.stop:
				return
			}
		}
	}()

	go func() {
		defer processor.running.Done()
		for {
			select {
			case t := <-processor.ticker:
				processor.sendHits(processor.takeHits(), t)
			case <-processor.stop:
				return
			}
		}
	}()

	return processor
}

// wait sleeps for the given duration, and reports whether the processor is still running.
func (d *DataCollectProcessor) wait(duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-d.stop:
		return false
	}
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

	return d.post(ctx, hits)
}

// post sends the hits as one request, splitting the batch in two when the body is over
// maxBatchBytes. Queue times are already computed at this point, so a split never recomputes them.
func (d *DataCollectProcessor) post(ctx context.Context, hits []map[string]interface{}) error {
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

	body, err := json.Marshal(batchHit)
	if err != nil {
		return fmt.Errorf("error when marshaling batch hit: %v", err)
	}

	if len(body) > d.maxBatchBytes && len(hits) > 1 {
		half := len(hits) / 2
		return errors.Join(d.post(ctx, hits[:half]), d.post(ctx, hits[half:]))
	}

	d.logger.Infof("sending %d hits to datacollect", len(hits))
	if err := d.doWithRetry(ctx, body); err != nil {
		return err
	}
	d.logger.Infof("%d hits sent to datacollect successfully", len(hits))

	return nil
}

// doWithRetry posts the body, retrying a failed attempt up to sendRetries times. Only transport
// errors and server errors are retried: a request the collector rejected would be rejected again.
//
// Delivery is therefore at-least-once: a request that timed out may have been received, and
// retrying it sends the batch twice. That is deliberate - losing hits is the worse outcome.
func (d *DataCollectProcessor) doWithRetry(ctx context.Context, body []byte) error {
	var err error
	for attempt := 0; attempt <= d.sendRetries; attempt++ {
		if attempt > 0 {
			d.logger.Warnf("retrying to send hits (attempt %d): %v", attempt, err)
			// Backing off follows ctx, not the processor lifetime: a shutdown still gets its
			// retries, bounded by the context the caller gave us.
			timer := time.NewTimer(time.Duration(attempt) * d.retryDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return err
			}
		}

		var retryable bool
		if retryable, err = d.do(ctx, body); err == nil || !retryable {
			return err
		}
	}

	return err
}

// do posts the body once, and reports whether a failure is worth retrying.
func (d *DataCollectProcessor) do(ctx context.Context, body []byte) (retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.trackingURL, bytes.NewBuffer(body))
	if err != nil {
		return false, fmt.Errorf("error when building the request: %v", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return true, fmt.Errorf("error when making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return true, fmt.Errorf("got status %v when calling HTTP request", resp.Status)
	}
	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("got status %v when calling HTTP request", resp.Status)
	}

	return false, nil
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

// takeHits detaches the pending hits from the processor and returns them.
// The caller owns the returned slice, so it can be sent without holding the lock.
func (d *DataCollectProcessor) takeHits() []models.MappableHit {
	d.lock.Lock()
	defer d.lock.Unlock()

	hits := d.hits
	d.hits = []models.MappableHit{}
	return hits
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

	// Append and detach under the same lock, so concurrent calls cannot send the same hits twice.
	var batch []models.MappableHit
	d.lock.Lock()
	d.hits = append(d.hits, mappableHits...)
	if len(d.hits) >= d.batchSize {
		batch, d.hits = d.hits, []models.MappableHit{}
		// Counted under the lock: Shutdown takes the lock before waiting, so a batch detached here
		// is always waited for.
		d.running.Add(1)
	}
	d.lock.Unlock()

	if len(batch) > 0 {
		go func() {
			defer d.running.Done()
			d.sendHits(batch, time.Now())
		}()
	}
	return nil
}

// Shutdown stops the batching goroutines, waits for the requests already started, then sends
// whatever is still pending. Hits are only dropped if ctx is cancelled first.
func (d *DataCollectProcessor) Shutdown(ctx context.Context) error {
	d.stopOnce.Do(func() { close(d.stop) })

	// Taking the lock once makes every batch detached by TrackHits visible to the wait below.
	d.lock.Lock()
	d.lock.Unlock() //nolint:staticcheck // barrier, not a guarded section

	done := make(chan struct{})
	go func() {
		d.running.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		// Out of time, but pending hits are still worth one attempt of their own.
		flushCtx, cancel := context.WithTimeout(context.Background(), d.httpClient.Timeout)
		defer cancel()
		_ = d.sendBatchHit(flushCtx, d.takeHits())
		return ctx.Err()
	}

	return d.sendBatchHit(ctx, d.takeHits())
}
