package assignments_managers

import (
	"sync"
	"time"

	common "github.com/flagship-io/flagship-common"
)

var lock = sync.Mutex{}
var cache = map[string]*common.VisitorAssignments{}
var separator = "."

type InMemory struct{}

func (*InMemory) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	lock.Lock()
	assignments := cache[envID+separator+visitorID]
	lock.Unlock()
	return assignments, nil
}

func (*InMemory) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	lock.Lock()
	cache[envID+separator+visitorID] = &common.VisitorAssignments{
		Timestamp:   date.UnixMilli(),
		Assignments: vgIDAssignments,
	}
	lock.Unlock()
	return nil
}
