package assignments_managers

import (
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	decision "github.com/flagship-io/flagship-common"
	"github.com/stretchr/testify/assert"
)

func TestMemoryCache(t *testing.T) {
	envID := "env_id"
	visID := "visID"
	m := &MemoryManager{}

	r, err := m.LoadAssignments(envID, visID)

	var nullResp *decision.VisitorAssignments
	assert.Nil(t, err)
	assert.Equal(t, nullResp, r)

	cache := &decision.VisitorAssignments{
		Timestamp:   time.Now().UnixMilli(),
		Assignments: make(map[string]*decision.VisitorCache),
	}
	cache.Assignments["vgID"] = &decision.VisitorCache{VariationID: "vID"}
	err = m.SaveAssignments(envID, visID, cache.Assignments, time.Now(), connectors.SaveAssignmentsContext{})

	assert.Equal(t, nil, err)

	r, err = m.LoadAssignments(envID, visID)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, r.Assignments["vgID"])
}
