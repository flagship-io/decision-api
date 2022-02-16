package assignments_managers

import (
	"encoding/json"
	"errors"
	"time"

	common "github.com/flagship-io/flagship-common"
	"github.com/prologic/bitcask"
)

// LocalCacheManager represents the local db manager object
type LocalCacheManager struct {
	db           *bitcask.Bitcask
	keySeparator string
}

// LocalOptions are the options necessary to make the local cache manager work
type LocalOptions struct {
	DbPath       string
	keySeparator string
}

func InitLocalCacheManager(localOptions LocalOptions) (m *LocalCacheManager, err error) {
	db, err := bitcask.Open(localOptions.DbPath)
	if err != nil {
		return nil, err
	}

	if localOptions.keySeparator == "" {
		localOptions.keySeparator = "."
	}
	m = &LocalCacheManager{
		db:           db,
		keySeparator: localOptions.keySeparator,
	}

	return m, nil
}

// Set saves the campaigns in cache for this visitor
func (m *LocalCacheManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	if m.db == nil {
		return errors.New("local cache manager not initialized")
	}

	cache, err := json.Marshal(&common.VisitorAssignments{
		Assignments: vgIDAssignments,
		Timestamp:   date.UnixMilli(),
	})

	if err == nil {
		err = m.db.Put([]byte(envID+m.keySeparator+visitorID), cache)
	}

	return err
}

// LoadAssignments returns the visitor assignment in cache
func (m *LocalCacheManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	if m.db == nil {
		return nil, errors.New("local cache manager not initialized")
	}

	data, err := m.db.Get([]byte(envID + m.keySeparator + visitorID))

	if err != nil {
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
func (m *LocalCacheManager) Dispose() error {
	if m.db == nil {
		return nil
	}
	return m.db.Close()
}
