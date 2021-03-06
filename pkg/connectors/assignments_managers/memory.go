package assignments_managers

import (
	"sync"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	common "github.com/flagship-io/flagship-common"
)

type MemoryManager struct {
	cache     map[string]*common.VisitorAssignments
	lock      *sync.Mutex
	separator string
}

func InitMemoryManager() *MemoryManager {
	return &MemoryManager{
		cache:     map[string]*common.VisitorAssignments{},
		lock:      &sync.Mutex{},
		separator: ".",
	}
}

func (m *MemoryManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	m.lock.Lock()
	assignments := m.cache[envID+m.separator+visitorID]
	m.lock.Unlock()
	return assignments, nil
}

func (d *MemoryManager) ShouldSaveAssignments(context connectors.SaveAssignmentsContext) bool {
	return true
}

func (m *MemoryManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	m.lock.Lock()
	assignments, ok := m.cache[envID+m.separator+visitorID]
	newAssignments := map[string]*common.VisitorCache{}
	if ok {
		for k, v := range assignments.Assignments {
			newAssignments[k] = v
		}
	}
	for k, v := range vgIDAssignments {
		newAssignments[k] = v
	}
	m.cache[envID+m.separator+visitorID] = &common.VisitorAssignments{
		Timestamp:   date.UnixMilli(),
		Assignments: newAssignments,
	}
	m.lock.Unlock()
	return nil
}
