package assignments_managers

import (
	"os"
	"testing"
	"time"

	decision "github.com/flagship-io/flagship-common"
	"github.com/stretchr/testify/assert"
)

func TestLocalCache(t *testing.T) {
	testFolder := "test"

	envID := "env_id"
	visID := "visID"
	notInitialized := &LocalManager{}
	_, err := notInitialized.LoadAssignments(envID, visID)
	assert.Equal(t, "local cache manager not initialized", err.Error())

	err = notInitialized.SaveAssignments(envID, visID, nil, time.Now())
	assert.Equal(t, "local cache manager not initialized", err.Error())

	m, err := InitLocalCacheManager(LocalOptions{
		DbPath: testFolder,
	})
	assert.Nil(t, err)

	r, err := m.LoadAssignments(envID, visID)

	var nullResp *decision.VisitorAssignments
	assert.Nil(t, err)
	assert.Equal(t, nullResp, r)

	cache := &decision.VisitorAssignments{
		Timestamp:   time.Now().UnixMilli(),
		Assignments: make(map[string]*decision.VisitorCache),
	}
	cache.Assignments["vgID"] = &decision.VisitorCache{VariationID: "vID"}
	err = m.SaveAssignments(envID, visID, cache.Assignments, time.Now())
	assert.Nil(t, err)

	r, err = m.LoadAssignments(envID, visID)
	assert.Nil(t, err)
	assert.Equal(t, "vID", r.Assignments["vgID"].VariationID)
	assert.Equal(t, false, r.Assignments["vgID"].Activated)

	cache.Assignments["vgID2"] = &decision.VisitorCache{VariationID: "vID2", Activated: true}
	err = m.SaveAssignments(envID, visID, cache.Assignments, time.Now())
	assert.Nil(t, err)

	r, err = m.LoadAssignments(envID, visID)
	assert.Nil(t, err)
	assert.Equal(t, "vID", r.Assignments["vgID"].VariationID)
	assert.Equal(t, "vID2", r.Assignments["vgID2"].VariationID)
	assert.Equal(t, true, r.Assignments["vgID2"].Activated)

	err = m.Dispose()
	assert.Nil(t, err)

	err = os.RemoveAll(testFolder)
	assert.Equal(t, nil, err)
}
