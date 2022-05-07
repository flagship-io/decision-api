package assignments_managers

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	common "github.com/flagship-io/flagship-common"
	"github.com/prologic/bitcask"
)

// LocalManager represents the local db manager object
type LocalManager struct {
	db           *bitcask.Bitcask
	keySeparator string
}

// LocalOptions are the options necessary to make the local cache manager work
type LocalOptions struct {
	DbPath       string
	keySeparator string
}

func InitLocalCacheManager(localOptions LocalOptions) (m *LocalManager, err error) {
	db, err := bitcask.Open(localOptions.DbPath)
	if err != nil {
		return nil, err
	}

	if localOptions.keySeparator == "" {
		localOptions.keySeparator = "."
	}
	m = &LocalManager{
		db:           db,
		keySeparator: localOptions.keySeparator,
	}

	return m, nil
}

// Set saves the campaigns in cache for this visitor
func (m *LocalManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time, context connectors.SaveAssignmentsContext) error {
	if m.db == nil {
		return errors.New("local cache manager not initialized")
	}

	key := []byte(envID + m.keySeparator + visitorID)

	assignments := make(map[string]*common.VisitorCache)
	if m.db.Has(key) {
		existing, err := m.LoadAssignments(envID, visitorID)
		if err != nil {
			return err
		}
		for k, v := range existing.Assignments {
			assignments[k] = v
		}
	}

	for k, v := range vgIDAssignments {
		assignments[k] = v
	}

	cache, err := json.Marshal(&common.VisitorAssignments{
		Assignments: assignments,
		Timestamp:   date.UnixMilli(),
	})

	if err == nil {
		err = m.db.Put(key, cache)
	}

	return err
}

// LoadAssignments returns the visitor assignment in cache
func (m *LocalManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	if m.db == nil {
		return nil, errors.New("local cache manager not initialized")
	}

	data, err := m.db.Get([]byte(envID + m.keySeparator + visitorID))

	if err != nil {
		if err == bitcask.ErrKeyNotFound {
			return nil, nil
		}
		return nil, err
	}

	assignments := &common.VisitorAssignments{}
	err = json.Unmarshal(data, &assignments)

	if err != nil {
		return nil, err
	}

	return assignments, nil
}

// Dispose frees IO resources
func (m *LocalManager) Dispose() error {
	if m.db == nil {
		return nil
	}
	return m.db.Close()
}
