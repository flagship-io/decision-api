package hits_processors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDataCollectBuilder(t *testing.T) {
	httpClient := &http.Client{}
	dc := NewDataCollectProcessor(
		WithBatchOptions(50, time.Second),
		WithLogger("debug", "json"),
		WithTrackingURL("https://tracking-url.dev"),
		WithHTTPClient(httpClient))

	assert.Equal(t, 50, dc.batchSize)
	assert.Equal(t, time.Second, dc.batchingWindow)
	assert.Equal(t, logrus.DebugLevel, dc.logger.Logger.Level)
	assert.Equal(t, "https://tracking-url.dev", dc.trackingURL)
	assert.Equal(t, httpClient, dc.httpClient)
}

func TestDataCollectTrack(t *testing.T) {
	lock := &sync.Mutex{}
	var bodySents []string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		lock.Lock()
		lastBodySent, _ := io.ReadAll(req.Body)
		bodySents = append(bodySents, string(lastBodySent))
		_, err := rw.Write([]byte("{}"))
		assert.Nil(t, err)
		lock.Unlock()
	}))
	// Close the server when test finishes
	defer server.Close()

	dcProcessor := NewDataCollectProcessor(WithBatchOptions(2, 100*time.Millisecond), WithTrackingURL(server.URL))
	ts := time.Now().Add(-1 * time.Second).UnixMilli()

	err := dcProcessor.TrackHits(connectors.TrackingHits{
		CampaignActivations: []*models.CampaignActivation{{
			EnvID:       "env_id",
			CustomerID:  "customer_id",
			VisitorID:   "visitor_id",
			CampaignID:  "campaign_id",
			VariationID: "variation_id",
			Timestamp:   ts,
		}},
		VisitorContext: []*models.VisitorContext{{
			EnvID:      "env_id",
			VisitorID:  "visitor_id",
			CustomerID: "customer_id",
			Partner:    "partner",
			Context:    map[string]interface{}{"key": "value"},
			Timestamp:  ts,
		}},
	})

	time.Sleep(110 * time.Millisecond)
	assert.Nil(t, err)
	lock.Lock()
	assert.Equal(t, 1, len(bodySents))

	batch := &batchHit{}
	err = json.Unmarshal([]byte(bodySents[0]), batch)
	lock.Unlock()
	assert.Nil(t, err)
	assert.Equal(t, "BATCH", batch.Type)
	assert.Equal(t, "APP", batch.DataSource)
	assert.Equal(t, 2, len(batch.Hits))
	assert.Equal(t, "env_id", batch.Hits[0]["cid"])
	assert.Equal(t, "customer_id", batch.Hits[0]["cuid"])
	assert.Equal(t, "visitor_id", batch.Hits[0]["vid"])
	assert.Equal(t, "CAMPAIGN", batch.Hits[0]["t"])
	assert.Equal(t, "campaign_id", batch.Hits[0]["caid"])
	assert.Equal(t, "variation_id", batch.Hits[0]["vaid"])
	assert.True(t, batch.Hits[0]["qt"].(float64) < 1010 && batch.Hits[0]["qt"].(float64) >= 1000)

	assert.Equal(t, "env_id", batch.Hits[1]["cid"])
	assert.Equal(t, "visitor_id", batch.Hits[1]["vid"])
	assert.Equal(t, "customer_id", batch.Hits[1]["cuid"])
	assert.Equal(t, "SEGMENT", batch.Hits[1]["t"])
	assert.EqualValues(t, map[string]interface{}{
		"key": "value",
	}, batch.Hits[1]["s"])
	assert.True(t, batch.Hits[1]["qt"].(float64) < 1010 && batch.Hits[1]["qt"].(float64) >= 1000)
}

// countingCollector records every hit id received, so a test can assert that none was lost
// and none was sent twice.
type countingCollector struct {
	lock          sync.Mutex
	server        *httptest.Server
	ids           []string
	requestCount  int
	largest       int
	delay         time.Duration
	failures      int
	failureStatus int
	userAgents    []string
	bodies        []string
}

func newCountingCollector() *countingCollector {
	c := &countingCollector{}
	c.server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)

		c.lock.Lock()
		c.requestCount++
		c.userAgents = append(c.userAgents, req.Header.Get("User-Agent"))
		c.bodies = append(c.bodies, string(body))
		if len(body) > c.largest {
			c.largest = len(body)
		}
		failing := c.failures > 0
		if failing {
			c.failures--
		}
		first := c.requestCount == 1
		delay := c.delay
		c.lock.Unlock()

		if failing {
			rw.WriteHeader(c.failureStatus)
			return
		}
		if first {
			time.Sleep(delay)
		}

		batch := &batchHit{}
		if err := json.Unmarshal(body, batch); err == nil {
			c.lock.Lock()
			for _, hit := range batch.Hits {
				id, _ := hit["vid"].(string)
				c.ids = append(c.ids, id)
			}
			c.lock.Unlock()
		}
		_, _ = rw.Write([]byte("{}"))
	}))
	return c
}

func (c *countingCollector) requests() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.requestCount
}

func (c *countingCollector) largestBody() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.largest
}

func (c *countingCollector) lastBody() string {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.bodies) == 0 {
		return ""
	}
	return c.bodies[len(c.bodies)-1]
}

func (c *countingCollector) lastUserAgent() string {
	c.lock.Lock()
	defer c.lock.Unlock()
	if len(c.userAgents) == 0 {
		return ""
	}
	return c.userAgents[len(c.userAgents)-1]
}

func (c *countingCollector) received() []string {
	c.lock.Lock()
	defer c.lock.Unlock()
	return append([]string{}, c.ids...)
}

// TestDataCollectTrackConcurrentlyLosesNoHit sends hits from many goroutines at once and checks
// that the collector receives every hit exactly once. Before the batching fix, concurrent callers
// could send the same pending hits several times, and hits appended during a flush were dropped.
func TestDataCollectTrackConcurrentlyLosesNoHit(t *testing.T) {
	collector := newCountingCollector()
	defer collector.server.Close()

	const callers, hitsPerCaller = 50, 20
	processor := NewDataCollectProcessor(
		WithBatchOptions(10, 50*time.Millisecond),
		WithTrackingURL(collector.server.URL))

	wg := &sync.WaitGroup{}
	for caller := 0; caller < callers; caller++ {
		wg.Add(1)
		go func(caller int) {
			defer wg.Done()
			for hit := 0; hit < hitsPerCaller; hit++ {
				err := processor.TrackHits(connectors.TrackingHits{
					VisitorContext: []*models.VisitorContext{{
						EnvID:     "env_id",
						VisitorID: fmt.Sprintf("visitor_%d_%d", caller, hit),
						Context:   map[string]interface{}{"key": "value"},
					}},
				})
				assert.Nil(t, err)
			}
		}(caller)
	}
	wg.Wait()

	assert.Nil(t, processor.Shutdown(context.Background()))
	assert.Eventually(t, func() bool {
		return len(collector.received()) >= callers*hitsPerCaller
	}, 5*time.Second, 20*time.Millisecond, "not every hit reached the collector")

	received := collector.received()
	seen := map[string]int{}
	for _, id := range received {
		seen[id]++
	}
	assert.Equal(t, callers*hitsPerCaller, len(received), "hits were duplicated or lost")
	assert.Equal(t, callers*hitsPerCaller, len(seen), "the same hit was sent more than once")
}

// TestDataCollectTrackDoesNotBlockOnBatchingWindow checks that TrackHits stays fast while the
// batching window elapses. The window used to be waited out while holding the lock TrackHits
// needs, which made every request pay for it.
func TestDataCollectTrackDoesNotBlockOnBatchingWindow(t *testing.T) {
	collector := newCountingCollector()
	defer collector.server.Close()

	window := 500 * time.Millisecond
	// A small batch size matters: it makes a size flush happen, which sets lastTick. Without it the
	// window is never actually waited out and the test would pass even on the unfixed code.
	processor := NewDataCollectProcessor(
		WithBatchOptions(10, window),
		WithTrackingURL(collector.server.URL))

	deadline := time.Now().Add(4 * window)
	slowest := time.Duration(0)
	for time.Now().Before(deadline) {
		start := time.Now()
		assert.Nil(t, processor.TrackHits(connectors.TrackingHits{
			VisitorContext: []*models.VisitorContext{{EnvID: "env_id", VisitorID: "visitor_id"}},
		}))
		if elapsed := time.Since(start); elapsed > slowest {
			slowest = elapsed
		}
		time.Sleep(time.Millisecond)
	}

	t.Logf("slowest TrackHits call: %v (batching window %v)", slowest, window)
	assert.Less(t, slowest, window/2, "TrackHits waited for the batching window")
}

// TestDataCollectShutdownDeliversEveryHit checks that no hit is lost when the process stops while a
// request is in flight: Shutdown must wait for it and flush whatever is still pending.
func TestDataCollectShutdownDeliversEveryHit(t *testing.T) {
	collector := newCountingCollector()
	// Only the first request is slow, so it is still in flight when Shutdown is called.
	collector.delay = 500 * time.Millisecond
	defer collector.server.Close()

	processor := NewDataCollectProcessor(
		WithBatchOptions(5, time.Hour),
		WithTrackingURL(collector.server.URL))

	for hit := 0; hit < 8; hit++ {
		assert.Nil(t, processor.TrackHits(connectors.TrackingHits{
			VisitorContext: []*models.VisitorContext{{
				EnvID:     "env_id",
				VisitorID: fmt.Sprintf("visitor_%d", hit),
			}},
		}))
	}

	// 5 hits are being sent by a slow collector, 3 are still pending.
	assert.Nil(t, processor.Shutdown(context.Background()))
	assert.Equal(t, 8, len(collector.received()), "Shutdown did not deliver every hit")
}

// TestDataCollectSplitsOversizedBatch checks that a batch too large for one request is split
// instead of being sent as a single body the collector may not be able to read in time.
func TestDataCollectSplitsOversizedBatch(t *testing.T) {
	collector := newCountingCollector()
	defer collector.server.Close()

	const hits = 40
	processor := NewDataCollectProcessor(
		WithBatchOptions(hits, time.Hour),
		WithSendOptions(2048),
		WithTrackingURL(collector.server.URL))

	for hit := 0; hit < hits; hit++ {
		assert.Nil(t, processor.TrackHits(connectors.TrackingHits{
			VisitorContext: []*models.VisitorContext{{
				EnvID:     "env_id",
				VisitorID: fmt.Sprintf("visitor_%d", hit),
				Context:   map[string]interface{}{"key": "a fairly long segment value to take room"},
			}},
		}))
	}
	assert.Nil(t, processor.Shutdown(context.Background()))

	assert.Equal(t, hits, len(collector.received()), "splitting lost or duplicated hits")
	assert.Greater(t, collector.requests(), 1, "the oversized batch was sent as a single request")
	assert.LessOrEqual(t, collector.largestBody(), 2048, "a request body went over the limit")
}

// TestDataCollectDoesNotRetryAFailedSend pins the decision to send each batch exactly once: a
// retry after a timeout can duplicate every hit in the batch, and a collector that is failing is
// not helped by being asked again.
func TestDataCollectDoesNotRetryAFailedSend(t *testing.T) {
	collector := newCountingCollector()
	collector.failures = 1
	collector.failureStatus = http.StatusServiceUnavailable
	defer collector.server.Close()

	processor := NewDataCollectProcessor(
		WithBatchOptions(1, time.Hour),
		WithTrackingURL(collector.server.URL))

	assert.Nil(t, processor.TrackHits(connectors.TrackingHits{
		VisitorContext: []*models.VisitorContext{{EnvID: "env_id", VisitorID: "visitor_id"}},
	}))
	assert.Nil(t, processor.Shutdown(context.Background()))

	assert.Equal(t, 1, collector.requests(), "a failed request was retried")
	assert.Equal(t, 0, len(collector.received()))
}

// TestDataCollectShutdownDeliversHitsFlushedByTheWindow covers the default path: a flush started by
// the batching window rather than by a full batch must also be waited for.
func TestDataCollectShutdownDeliversHitsFlushedByTheWindow(t *testing.T) {
	collector := newCountingCollector()
	collector.delay = 500 * time.Millisecond
	defer collector.server.Close()

	processor := NewDataCollectProcessor(
		WithBatchOptions(1000, 50*time.Millisecond),
		WithTrackingURL(collector.server.URL))

	for hit := 0; hit < 5; hit++ {
		assert.Nil(t, processor.TrackHits(connectors.TrackingHits{
			VisitorContext: []*models.VisitorContext{{
				EnvID:     "env_id",
				VisitorID: fmt.Sprintf("visitor_%d", hit),
			}},
		}))
	}

	// Let the batching window start a flush, then stop while it is still in flight.
	time.Sleep(150 * time.Millisecond)
	assert.Nil(t, processor.Shutdown(context.Background()))
	assert.Equal(t, 5, len(collector.received()), "Shutdown did not wait for the periodic flush")
}

// TestDataCollectShutdownFlushesWhatItCanWhenOutOfTime covers the branch taken when a request
// already in flight outlives the shutdown budget. The hits still buffered are not that request's
// problem, and dropping them would lose data the collector never saw.
func TestDataCollectShutdownFlushesWhatItCanWhenOutOfTime(t *testing.T) {
	collector := newCountingCollector()
	collector.delay = time.Second
	defer collector.server.Close()

	processor := NewDataCollectProcessor(
		WithBatchOptions(2, time.Hour),
		WithTrackingURL(collector.server.URL))

	// The first two hits fill the batch and start a send that will not come back in time.
	// The third stays buffered, and is what Shutdown has to rescue.
	for hit := 0; hit < 3; hit++ {
		assert.Nil(t, processor.TrackHits(connectors.TrackingHits{
			VisitorContext: []*models.VisitorContext{{
				EnvID:     "env_id",
				VisitorID: fmt.Sprintf("visitor_%d", hit),
			}},
		}))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := processor.Shutdown(ctx)

	assert.ErrorIs(t, err, context.DeadlineExceeded, "Shutdown should report that it ran out of time")
	assert.Equal(t, []string{"visitor_2"}, collector.received(),
		"the buffered hit was dropped instead of being flushed")
}

// TestDataCollectIdentifiesItself checks that a request says which software sent it, and that the
// batch repeats it in a field placed before the hits so a truncated body still carries it.
func TestDataCollectIdentifiesItself(t *testing.T) {
	collector := newCountingCollector()
	defer collector.server.Close()

	processor := NewDataCollectProcessor(
		WithBatchOptions(1, time.Hour),
		WithTrackingURL(collector.server.URL))

	assert.Nil(t, processor.TrackHits(connectors.TrackingHits{
		VisitorContext: []*models.VisitorContext{{EnvID: "env_id", VisitorID: "visitor_id"}},
	}))
	assert.Nil(t, processor.Shutdown(context.Background()))

	assert.Contains(t, collector.lastUserAgent(), "flagship-decision-api/")
	assert.Contains(t, collector.lastUserAgent(), "self-hosted")
	assert.Contains(t, collector.lastUserAgent(), runtime.Version())

	body := collector.lastBody()
	assert.Contains(t, body, `"runner, self-hosted"`)
	assert.Less(t, strings.Index(body, `"cv"`), strings.Index(body, `"h"`),
		"cv must come before h, so a truncated body still identifies the sender")
}

// TestVersionReportsTheBuild covers the other half of version(): a released binary carries a tag,
// and the User-Agent has to report it rather than the placeholder.
func TestVersionReportsTheBuild(t *testing.T) {
	original := models.Version
	defer func() { models.Version = original }()

	models.Version = "v1.2.3"
	assert.Equal(t, "v1.2.3", version())

	models.Version = ""
	assert.Equal(t, "unknown", version(), "a build with no version stamped must say so")
}
