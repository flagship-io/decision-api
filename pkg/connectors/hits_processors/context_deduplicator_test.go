package hits_processors

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	"github.com/stretchr/testify/assert"
)

// recorder counts what reaches the wrapped processor.
type recorder struct {
	lock      sync.Mutex
	calls     int
	contexts  []*models.VisitorContext
	shutdowns int
}

func (r *recorder) TrackHits(hits connectors.TrackingHits) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.calls++
	r.contexts = append(r.contexts, hits.VisitorContext...)
	return nil
}

func (r *recorder) Shutdown(context.Context) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.shutdowns++
	return errors.New("from the wrapped processor")
}

func (r *recorder) counts() (calls int, contexts int) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.calls, len(r.contexts)
}

func newTestDeduplicator(next connectors.HitsProcessor) *ContextDeduplicator {
	return NewContextDeduplicator(next, 1000, time.Hour, "error", logger.FORMAT_TEXT)
}

func visitorContext(visitorID string, ctx map[string]interface{}) *models.VisitorContext {
	return &models.VisitorContext{
		EnvID:     "env_id",
		VisitorID: visitorID,
		Context:   ctx,
		Timestamp: time.Now().UnixMilli(),
	}
}

func track(t *testing.T, d *ContextDeduplicator, contexts ...*models.VisitorContext) {
	t.Helper()
	assert.NoError(t, d.TrackHits(connectors.TrackingHits{VisitorContext: contexts}))
}

func TestContextDeduplicatorSuppressesAnUnchangedContext(t *testing.T) {
	next := &recorder{}
	d := newTestDeduplicator(next)

	for i := 0; i < 8; i++ {
		track(t, d, visitorContext("visitor", map[string]interface{}{"plan": "premium", "age": 42}))
	}

	calls, contexts := next.counts()
	assert.Equal(t, 1, contexts, "only the first context should have been sent")
	assert.Equal(t, 1, calls, "a call with nothing left to send should not reach the processor at all")
}

func TestContextDeduplicatorSendsEveryChange(t *testing.T) {
	cases := []struct {
		name  string
		first *models.VisitorContext
		then  *models.VisitorContext
	}{
		{
			name:  "a value changes",
			first: visitorContext("visitor", map[string]interface{}{"plan": "premium"}),
			then:  visitorContext("visitor", map[string]interface{}{"plan": "free"}),
		},
		{
			name:  "a key is added",
			first: visitorContext("visitor", map[string]interface{}{"plan": "premium"}),
			then:  visitorContext("visitor", map[string]interface{}{"plan": "premium", "age": 42}),
		},
		{
			name:  "a key is removed",
			first: visitorContext("visitor", map[string]interface{}{"plan": "premium", "age": 42}),
			then:  visitorContext("visitor", map[string]interface{}{"plan": "premium"}),
		},
		{
			name:  "a value changes type",
			first: visitorContext("visitor", map[string]interface{}{"premium": true}),
			then:  visitorContext("visitor", map[string]interface{}{"premium": "true "}),
		},
		{
			// Without length framing, "ab"+"c" and "a"+"bc" hash the same.
			name:  "a key and its value swap a character",
			first: visitorContext("visitor", map[string]interface{}{"ab": "c"}),
			then:  visitorContext("visitor", map[string]interface{}{"a": "bc"}),
		},
		{
			name:  "another visitor sends the same context",
			first: visitorContext("visitor", map[string]interface{}{"plan": "premium"}),
			then:  visitorContext("other_visitor", map[string]interface{}{"plan": "premium"}),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next := &recorder{}
			d := newTestDeduplicator(next)

			track(t, d, c.first)
			track(t, d, c.then)

			_, contexts := next.counts()
			assert.Equal(t, 2, contexts)
		})
	}
}

func TestContextDeduplicatorSendsAgainWhenTheVisitorIsIdentified(t *testing.T) {
	next := &recorder{}
	d := newTestDeduplicator(next)

	anonymous := visitorContext("anonymous_id", map[string]interface{}{"plan": "premium"})
	identified := visitorContext("anonymous_id", map[string]interface{}{"plan": "premium"})
	identified.CustomerID = "customer_id"

	track(t, d, anonymous)
	track(t, d, identified)

	_, contexts := next.counts()
	assert.Equal(t, 2, contexts, "logging in links the visitor to a customer and has to be sent")
}

func TestContextDeduplicatorKeepsPartnersApart(t *testing.T) {
	next := &recorder{}
	d := newTestDeduplicator(next)

	own := visitorContext("visitor", map[string]interface{}{"plan": "premium"})
	partner := visitorContext("visitor", map[string]interface{}{"plan": "premium"})
	partner.Partner = "mixpanel"

	track(t, d, own, partner)
	track(t, d, own, partner)

	_, contexts := next.counts()
	assert.Equal(t, 2, contexts, "the two are separate hits and must be tracked separately")
}

func TestContextDeduplicatorSendsAgainAfterTheTTL(t *testing.T) {
	next := &recorder{}
	d := NewContextDeduplicator(next, 1000, minContextTTL, "error", logger.FORMAT_TEXT)

	track(t, d, visitorContext("visitor", map[string]interface{}{"plan": "premium"}))
	track(t, d, visitorContext("visitor", map[string]interface{}{"plan": "premium"}))

	// Age the entry rather than waiting a minute for it.
	d.lock.Lock()
	for k, entry := range d.current {
		d.current[k] = contextEntry{hash: entry.hash, sentAt: entry.sentAt.Add(-2 * minContextTTL)}
	}
	d.lock.Unlock()

	track(t, d, visitorContext("visitor", map[string]interface{}{"plan": "premium"}))

	_, contexts := next.counts()
	assert.Equal(t, 2, contexts, "a visitor still active keeps a heartbeat, one hit per TTL")
}

func TestContextDeduplicatorAlwaysForwardsActivations(t *testing.T) {
	next := &recorder{}
	d := newTestDeduplicator(next)

	hits := connectors.TrackingHits{
		CampaignActivations: []*models.CampaignActivation{{EnvID: "env_id", VisitorID: "visitor"}},
		VisitorContext:      []*models.VisitorContext{visitorContext("visitor", map[string]interface{}{"plan": "premium"})},
	}
	assert.NoError(t, d.TrackHits(hits))
	assert.NoError(t, d.TrackHits(hits))

	calls, contexts := next.counts()
	assert.Equal(t, 2, calls, "an activation is sent once per assignment and is never deduplicated")
	assert.Equal(t, 1, contexts)
}

func TestContextDeduplicatorLeavesTheCallersHitsAlone(t *testing.T) {
	next := &recorder{}
	d := newTestDeduplicator(next)

	unchanged := visitorContext("visitor", map[string]interface{}{"plan": "premium"})
	changing := visitorContext("other_visitor", map[string]interface{}{"plan": "premium"})
	track(t, d, unchanged, changing)

	// The first is now suppressed and the second is not, so a filter writing over its input would
	// move the second hit into the first one's place.
	changing.Context["plan"] = "free"
	hits := connectors.TrackingHits{VisitorContext: []*models.VisitorContext{unchanged, changing}}
	assert.NoError(t, d.TrackHits(hits))

	assert.Equal(t, []*models.VisitorContext{unchanged, changing}, hits.VisitorContext,
		"the caller's slice must survive being filtered")
}

func TestContextDeduplicatorStaysBounded(t *testing.T) {
	next := &recorder{}
	d := NewContextDeduplicator(next, 100, time.Hour, "error", logger.FORMAT_TEXT)

	for i := 0; i < 10_000; i++ {
		track(t, d, visitorContext(fmt.Sprintf("visitor_%d", i), map[string]interface{}{"plan": "premium"}))
	}

	d.lock.Lock()
	defer d.lock.Unlock()
	assert.LessOrEqual(t, len(d.current)+len(d.previous), 2*100)
}

func TestContextDeduplicatorRefusesUnusableOptions(t *testing.T) {
	// A TTL written without a unit is read as nanoseconds and would deduplicate nothing.
	d := NewContextDeduplicator(&recorder{}, 0, 30, "error", logger.FORMAT_TEXT)

	assert.Equal(t, DefaultContextTTL, d.ttl)
	assert.Equal(t, DefaultContextMaxEntries, d.maxEntries)
}

func TestContextDeduplicatorForwardsShutdown(t *testing.T) {
	next := &recorder{}
	d := newTestDeduplicator(next)

	err := d.Shutdown(context.Background())

	assert.EqualError(t, err, "from the wrapped processor")
	assert.Equal(t, 1, next.shutdowns)
}

func TestContextDeduplicatorUnderConcurrentLoad(t *testing.T) {
	next := &recorder{}
	d := NewContextDeduplicator(next, 50, time.Hour, "error", logger.FORMAT_TEXT)

	wg := &sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				track(t, d, visitorContext(fmt.Sprintf("visitor_%d", i*20+j), map[string]interface{}{"plan": "premium"}))
			}
		}(i)
	}
	wg.Wait()

	_, contexts := next.counts()
	assert.Equal(t, 1000, contexts, "every distinct visitor is sent exactly once")
}

func TestContextDeduplicatorSuppressesFromTheOlderGeneration(t *testing.T) {
	next := &recorder{}
	// Two visitors per generation, so the third one below forces a rotation.
	d := NewContextDeduplicator(next, 2, time.Hour, "error", logger.FORMAT_TEXT)

	returning := visitorContext("returning_visitor", map[string]interface{}{"plan": "premium"})
	track(t, d, returning)
	track(t, d, visitorContext("filler_1", map[string]interface{}{"plan": "premium"}))
	track(t, d, visitorContext("filler_2", map[string]interface{}{"plan": "premium"}))

	d.lock.Lock()
	_, inOlderGeneration := d.previous[returning.EnvID+contextKeySeparator+returning.VisitorID+contextKeySeparator]
	d.lock.Unlock()
	assert.True(t, inOlderGeneration, "the visitor should have been demoted, or this test proves nothing")

	track(t, d, returning)

	_, contexts := next.counts()
	assert.Equal(t, 3, contexts, "a visitor in the older generation is still deduplicated")
}

// failing stands in for a wrapped processor that cannot take the hits.
type failing struct{ recorder }

func (f *failing) TrackHits(hits connectors.TrackingHits) error {
	_ = f.recorder.TrackHits(hits)
	return errors.New("collector unreachable")
}

func TestContextDeduplicatorForgetsAContextItCouldNotSend(t *testing.T) {
	next := &failing{}
	d := newTestDeduplicator(next)

	assert.Error(t, d.TrackHits(connectors.TrackingHits{
		VisitorContext: []*models.VisitorContext{visitorContext("visitor", map[string]interface{}{"plan": "premium"})},
	}))
	assert.Error(t, d.TrackHits(connectors.TrackingHits{
		VisitorContext: []*models.VisitorContext{visitorContext("visitor", map[string]interface{}{"plan": "premium"})},
	}))

	_, contexts := next.counts()
	assert.Equal(t, 2, contexts, "a context that never reached the collector must be offered again")
}
