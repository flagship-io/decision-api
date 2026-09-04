package hits_processors

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

// DefaultContextTTL is how long a visitor context stays deduplicated. After it, the next decision
// call sends the context again even if nothing changed, so reporting keeps a heartbeat per visitor.
const DefaultContextTTL = 30 * time.Minute

// minContextTTL is the shortest TTL accepted, for the same reason as minBatchingWindow: a duration
// written without a unit is read as nanoseconds, and a TTL of "30" would deduplicate nothing while
// looking like it was configured.
const minContextTTL = time.Minute

// DefaultContextMaxEntries is how many visitors a generation remembers before the older generation
// is dropped. An entry is a key plus a hash and a date, so the two generations together cost a few
// tens of megabytes at this default.
const DefaultContextMaxEntries = 100_000

// contextKeySeparator separates the parts of a key. A visitor ID arrives as a protobuf string and
// could contain a NUL itself, so a visitor can build an ID that makes their own request-body context
// share a key with their own partner context. Both are that visitor's own hits, and the environment
// and visitor ID come first, so no visitor can reach another one's entry.
const contextKeySeparator = "\x00"

const contextLogName = "Context Deduplicator"

type contextEntry struct {
	hash   uint64
	sentAt time.Time
}

// ContextDeduplicator wraps a hits processor and drops the visitor context hits that repeat a
// context already sent for that visitor. The decision API sends one on every call, so a visitor
// whose context never changes produces one identical hit per call; only the first is useful.
//
// It remembers visitors in two generations. The newer one is retired once it holds maxEntries
// visitors and the older one is dropped, so the process holds at most twice maxEntries whatever the
// traffic. Each entry also carries the date it was sent, and stops counting after the TTL.
//
// A visitor is forgotten two rotations after it was last sent, whether or not it kept coming back:
// entries expire from when they were written, not from when they were last read, which is what
// leaves each visitor a heartbeat rather than one hit for a whole day of activity. Size maxEntries
// above the visitors seen in a TTL, or generations rotate faster than the TTL and the cache spends
// its time forgetting the very visitors it is meant to deduplicate.
//
// Nothing here is correctness-bearing: forgetting a visitor sends a hit that would have been sent
// anyway. A restart, or a second replica, costs one extra hit per visitor. That is why the state
// stays in memory rather than in the assignment cache - it needs neither durability nor sharing.
//
// This is on by default. What a visitor was segmented on is unchanged, because every distinct
// context still reaches the collector at least once per TTL - only the number of times an unchanged
// one is repeated goes down. Reporting that counts visitors, or the values they were segmented on,
// sees the same thing; reporting that counts raw hits does not. Set hits.deduplicate_context to
// false to turn it off.
type ContextDeduplicator struct {
	next       connectors.HitsProcessor
	current    map[string]contextEntry
	previous   map[string]contextEntry
	lock       sync.Mutex
	maxEntries int
	ttl        time.Duration
	logger     *logger.Logger
	suppressed int64
	sent       int64
}

// NewContextDeduplicator wraps next so that repeated visitor contexts are not forwarded to it.
// Campaign activations are always forwarded: they are one per assignment, not one per call.
func NewContextDeduplicator(next connectors.HitsProcessor, maxEntries int, ttl time.Duration, lvl string, format logger.LogFormat) *ContextDeduplicator {
	log := logger.New(lvl, format, contextLogName)

	if maxEntries <= 0 {
		log.Warnf("context cache size %d is not usable, using %d", maxEntries, DefaultContextMaxEntries)
		maxEntries = DefaultContextMaxEntries
	}
	if ttl < minContextTTL {
		log.Warnf("context TTL %s is shorter than %s, using %s. Durations need a unit, for instance 30m",
			ttl, minContextTTL, DefaultContextTTL)
		ttl = DefaultContextTTL
	}

	log.Infof("deduplicating visitor context hits for %s, up to %d visitors", ttl, maxEntries)

	return &ContextDeduplicator{
		next:       next,
		current:    map[string]contextEntry{},
		previous:   map[string]contextEntry{},
		maxEntries: maxEntries,
		ttl:        ttl,
		logger:     log,
	}
}

func (d *ContextDeduplicator) TrackHits(hits connectors.TrackingHits) error {
	// hits is a copy, so replacing the slice leaves the caller's own untouched.
	kept, remembered := d.keep(hits.VisitorContext)
	hits.VisitorContext = kept

	if len(hits.VisitorContext) == 0 && len(hits.CampaignActivations) == 0 {
		return nil
	}

	err := d.next.TrackHits(hits)
	if err != nil {
		// The hits never left, so the visitors must not be remembered as having sent them, or the
		// next call would suppress a context that has never reached the collector.
		d.forget(remembered)
	}

	return err
}

// forget drops what keep remembered, so those contexts are sent again on the next call.
func (d *ContextDeduplicator) forget(keys []string) {
	if len(keys) == 0 {
		return
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	for _, key := range keys {
		delete(d.current, key)
		delete(d.previous, key)
	}
}

func (d *ContextDeduplicator) Shutdown(ctx context.Context) error {
	d.lock.Lock()
	d.logger.Infof("%d visitor context hits suppressed, %d sent", d.suppressed, d.sent)
	d.lock.Unlock()

	return d.next.Shutdown(ctx)
}

// keep returns the contexts worth sending - the ones not sent for this visitor yet, the ones whose
// content changed since they were, and the ones last sent longer ago than the TTL - along with the
// keys it remembered for them, so the caller can undo that if the send fails.
func (d *ContextDeduplicator) keep(contexts []*models.VisitorContext) (kept []*models.VisitorContext, remembered []string) {
	if len(contexts) == 0 {
		return contexts, nil
	}

	// Hashing and building keys is the expensive part, so it happens before the lock is taken.
	keys := make([]string, len(contexts))
	hashes := make([]uint64, len(contexts))
	for i, c := range contexts {
		keys[i] = c.EnvID + contextKeySeparator + c.VisitorID + contextKeySeparator + c.Partner
		hashes[i] = hashContext(c)
	}

	kept = make([]*models.VisitorContext, 0, len(contexts))
	now := time.Now()

	d.lock.Lock()
	defer d.lock.Unlock()

	for i, c := range contexts {
		entry, known := d.current[keys[i]]
		if !known {
			// The older generation still counts, or a visitor would start sending again every time
			// the newer one fills up.
			entry, known = d.previous[keys[i]]
		}

		if known && entry.hash == hashes[i] && now.Sub(entry.sentAt) < d.ttl {
			d.suppressed++
			continue
		}

		d.rotate()
		d.current[keys[i]] = contextEntry{hash: hashes[i], sentAt: now}
		d.sent++
		kept = append(kept, c)
		remembered = append(remembered, keys[i])
	}

	return kept, remembered
}

// rotate retires the current generation once it is full. The caller must hold the lock.
func (d *ContextDeduplicator) rotate() {
	if len(d.current) < d.maxEntries {
		return
	}

	d.logger.Infof("context cache full at %d visitors, %d hits suppressed so far, %d sent",
		len(d.current), d.suppressed, d.sent)
	d.previous = d.current
	d.current = map[string]contextEntry{}
}

// hashContext fingerprints everything about a context hit that ends up on the wire apart from its
// timestamp. Keys are sorted so map iteration order does not matter, and each part is hashed with
// its length so that "ab" followed by "c" cannot look like "a" followed by "bc".
func hashContext(c *models.VisitorContext) uint64 {
	keys := make([]string, 0, len(c.Context))
	for k := range c.Context {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	// The customer ID is sent alongside the context and links an anonymous visitor to a known one,
	// so a visitor logging in has to send its context again.
	h := hashPart(fnvOffset, c.CustomerID)
	for _, k := range keys {
		h = hashPart(h, k)
		// Values are compared as they will be sent, so a change that ToMap would flatten away does
		// not count as a change here either.
		h = hashPart(h, contextValue(c.Context[k]))
	}

	return h
}

// FNV-1a, written out rather than taken from hash/fnv because that package only accepts []byte:
// every key and every value would be copied into a fresh slice to be hashed, on every decision call.
// Sorting the keys still allocates, so this is cheaper hashing, not free hashing.
const (
	fnvOffset uint64 = 14695981039346656037
	fnvPrime  uint64 = 1099511628211
)

func hashPart(h uint64, part string) uint64 {
	h = (h ^ uint64(len(part))) * fnvPrime
	for i := 0; i < len(part); i++ {
		h = (h ^ uint64(part[i])) * fnvPrime
	}
	return h
}

// contextValue renders a context value the way the hit will. Most are already strings, and going
// through fmt for those would allocate a copy of every value on every decision call.
func contextValue(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}
