package assignments_managers

import (
	"sync"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	common "github.com/flagship-io/flagship-common"
)

// DefaultMemoryMaxEntries is how many visitors a generation of the in-memory cache holds before the
// older generation is dropped.
const DefaultMemoryMaxEntries = 100_000

// DefaultMemoryTTL is how long a generation lives when it does not fill up first. It matches the
// redis manager's default, so switching between them does not change how long a visitor keeps its
// variation.
const DefaultMemoryTTL = 3 * 30 * 24 * time.Hour

// MemoryManager keeps visitor assignments in memory, in two generations. Writes and reads both go
// to the newer generation, which is retired once it holds maxEntries visitors or once it has lived
// for the whole TTL; the older one is then dropped. The process therefore holds at most twice
// maxEntries visitors whatever the traffic - so pick maxEntries for the memory you can spare, since
// an entry grows with the number of campaigns a visitor has been assigned to (a few hundred bytes
// per campaign).
//
// A visitor is only forgotten after two rotations without being seen, since reading an assignment
// moves it back into the newer generation. Assignments are still lost when the process stops: use
// the redis or dynamo manager when they have to survive a restart.
type MemoryManager struct {
	current    map[string]*common.VisitorAssignments
	previous   map[string]*common.VisitorAssignments
	rotatedAt  time.Time
	lock       *sync.Mutex
	separator  string
	maxEntries int
	ttl        time.Duration
}

func InitMemoryManager() *MemoryManager {
	return InitMemoryManagerWithOptions(DefaultMemoryMaxEntries, DefaultMemoryTTL)
}

// InitMemoryManagerWithOptions builds a manager whose generations hold at most maxEntries visitors
// and live at most ttl.
func InitMemoryManagerWithOptions(maxEntries int, ttl time.Duration) *MemoryManager {
	if maxEntries <= 0 {
		maxEntries = DefaultMemoryMaxEntries
	}
	if ttl <= 0 {
		ttl = DefaultMemoryTTL
	}

	return &MemoryManager{
		current:    map[string]*common.VisitorAssignments{},
		previous:   map[string]*common.VisitorAssignments{},
		rotatedAt:  time.Now(),
		lock:       &sync.Mutex{},
		separator:  ".",
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

func (m *MemoryManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	key := envID + m.separator + visitorID

	m.lock.Lock()
	defer m.lock.Unlock()

	if assignments, ok := m.current[key]; ok {
		return assignments, nil
	}

	// A visitor who keeps coming back is usually only read: the caller saves an assignment when it
	// makes a new one, not on every decision. Reading therefore has to keep the visitor alive, or a
	// returning visitor would lose its variation two rotations after being assigned.
	assignments, ok := m.previous[key]
	if !ok {
		return nil, nil
	}

	m.rotate()
	m.current[key] = assignments
	return assignments, nil
}

func (d *MemoryManager) ShouldSaveAssignments(context connectors.SaveAssignmentsContext) bool {
	return true
}

func (m *MemoryManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	key := envID + m.separator + visitorID

	m.lock.Lock()
	defer m.lock.Unlock()

	newAssignments := map[string]*common.VisitorCache{}
	for _, generation := range []map[string]*common.VisitorAssignments{m.current, m.previous} {
		if assignments, ok := generation[key]; ok {
			for k, v := range assignments.Assignments {
				newAssignments[k] = v
			}
			break
		}
	}

	for k, v := range vgIDAssignments {
		newAssignments[k] = v
	}

	m.rotate()
	m.current[key] = &common.VisitorAssignments{
		Timestamp:   date.UnixMilli(),
		Assignments: newAssignments,
	}
	return nil
}

// rotate retires the current generation once it is full or has lived long enough. The caller must
// hold the lock.
func (m *MemoryManager) rotate() {
	if len(m.current) < m.maxEntries && time.Since(m.rotatedAt) < m.ttl {
		return
	}

	m.previous = m.current
	m.current = map[string]*common.VisitorAssignments{}
	m.rotatedAt = time.Now()
}
