package assignments_managers

import (
	"sync"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	common "github.com/flagship-io/flagship-common"
)

var lock = sync.Mutex{}
var cache = map[string]*common.VisitorAssignments{}
var separator = "."

type MemoryManager struct{}

func (*MemoryManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	lock.Lock()
	assignments := cache[envID+separator+visitorID]
	lock.Unlock()
	return assignments, nil
}

func (*MemoryManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time, context connectors.SaveAssignmentsContext) error {
	lock.Lock()
	cache[envID+separator+visitorID] = &common.VisitorAssignments{
		Timestamp:   date.UnixMilli(),
		Assignments: vgIDAssignments,
	}
	lock.Unlock()
	return nil
}
