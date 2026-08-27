package assignments_managers

import (
	"fmt"
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	decision "github.com/flagship-io/flagship-common"
	"github.com/stretchr/testify/assert"
)

func TestMemoryCache(t *testing.T) {
	envID := "env_id"
	visID := "visID"
	m := InitMemoryManager()
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

	assert.Equal(t, nil, err)

	r, err = m.LoadAssignments(envID, visID)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, r.Assignments["vgID"])

	cache.Assignments["vgID2"] = &decision.VisitorCache{VariationID: "vID2", Activated: true}
	err = m.SaveAssignments(envID, visID, cache.Assignments, time.Now())

	assert.Equal(t, nil, err)

	r, err = m.LoadAssignments(envID, visID)
	assert.Equal(t, nil, err)
	assert.Equal(t, "vID", r.Assignments["vgID"].VariationID)
	assert.Equal(t, "vID2", r.Assignments["vgID2"].VariationID)
	assert.Equal(t, true, r.Assignments["vgID2"].Activated)

	shouldSaveAssignments := m.ShouldSaveAssignments(connectors.SaveAssignmentsContext{
		AssignmentScope: connectors.Decision,
	})
	assert.True(t, shouldSaveAssignments)
	shouldSaveAssignments = m.ShouldSaveAssignments(connectors.SaveAssignmentsContext{
		AssignmentScope: connectors.Activation,
	})
	assert.True(t, shouldSaveAssignments)
}

// TestMemoryManagerStaysBounded checks that the cache stops growing instead of holding every
// visitor the process has ever seen.
func TestMemoryManagerStaysBounded(t *testing.T) {
	manager := InitMemoryManagerWithOptions(100, time.Hour)

	for visitor := 0; visitor < 10_000; visitor++ {
		err := manager.SaveAssignments("env_id", fmt.Sprintf("visitor_%d", visitor),
			map[string]*decision.VisitorCache{"vg": {VariationID: "v"}}, time.Now())
		assert.Nil(t, err)
	}

	assert.LessOrEqual(t, len(manager.current)+len(manager.previous), 200,
		"the cache kept growing with the number of visitors")
}

// TestMemoryManagerKeepsRecentAssignments checks that bounding the cache does not cost a visitor
// its assignment straight away: the previous generation is still readable.
func TestMemoryManagerKeepsRecentAssignments(t *testing.T) {
	manager := InitMemoryManagerWithOptions(10, time.Hour)

	assert.Nil(t, manager.SaveAssignments("env_id", "visitor_id",
		map[string]*decision.VisitorCache{"vg": {VariationID: "variation_id"}}, time.Now()))

	// Fill the current generation so it is retired, and the visitor moves to the previous one.
	for visitor := 0; visitor < 10; visitor++ {
		assert.Nil(t, manager.SaveAssignments("env_id", fmt.Sprintf("other_%d", visitor),
			map[string]*decision.VisitorCache{"vg": {VariationID: "v"}}, time.Now()))
	}

	assignments, err := manager.LoadAssignments("env_id", "visitor_id")
	assert.Nil(t, err)
	assert.NotNil(t, assignments, "the assignment was dropped as soon as the generation rotated")
	assert.Equal(t, "variation_id", assignments.Assignments["vg"].VariationID)
}

// TestMemoryManagerKeepsVisitorsThatKeepComingBack covers the case the cache exists for: a visitor
// who is assigned once and then only read must keep its variation. The caller saves an assignment
// when it makes a new one, not on every decision, so reads have to keep the visitor alive.
func TestMemoryManagerKeepsVisitorsThatKeepComingBack(t *testing.T) {
	manager := InitMemoryManagerWithOptions(10, time.Hour)

	assert.Nil(t, manager.SaveAssignments("env_id", "visitor_id",
		map[string]*decision.VisitorCache{"vg": {VariationID: "variation_id"}}, time.Now()))

	// Four generations go by, with the visitor only ever being read.
	for generation := 0; generation < 4; generation++ {
		for visitor := 0; visitor < 10; visitor++ {
			assert.Nil(t, manager.SaveAssignments("env_id", fmt.Sprintf("other_%d_%d", generation, visitor),
				map[string]*decision.VisitorCache{"vg": {VariationID: "v"}}, time.Now()))
		}

		assignments, err := manager.LoadAssignments("env_id", "visitor_id")
		assert.Nil(t, err)
		assert.NotNil(t, assignments, "a returning visitor lost its assignment after %d rotations", generation+1)
		assert.Equal(t, "variation_id", assignments.Assignments["vg"].VariationID)
	}
}

// TestMemoryManagerForgetsVisitorsThatStopComingBack is the other half: a visitor that is never
// seen again does get dropped, which is what keeps the cache bounded.
func TestMemoryManagerForgetsVisitorsThatStopComingBack(t *testing.T) {
	manager := InitMemoryManagerWithOptions(10, time.Hour)

	assert.Nil(t, manager.SaveAssignments("env_id", "visitor_id",
		map[string]*decision.VisitorCache{"vg": {VariationID: "variation_id"}}, time.Now()))

	for visitor := 0; visitor < 30; visitor++ {
		assert.Nil(t, manager.SaveAssignments("env_id", fmt.Sprintf("other_%d", visitor),
			map[string]*decision.VisitorCache{"vg": {VariationID: "v"}}, time.Now()))
	}

	assignments, err := manager.LoadAssignments("env_id", "visitor_id")
	assert.Nil(t, err)
	assert.Nil(t, assignments)
}

// TestMemoryManagerRotatesOnTTL checks that a generation is retired on age as well as on size.
func TestMemoryManagerRotatesOnTTL(t *testing.T) {
	manager := InitMemoryManagerWithOptions(1000, time.Millisecond)

	assert.Nil(t, manager.SaveAssignments("env_id", "visitor_id",
		map[string]*decision.VisitorCache{"vg": {VariationID: "variation_id"}}, time.Now()))
	assert.Equal(t, 1, len(manager.current))

	time.Sleep(2 * time.Millisecond)
	assert.Nil(t, manager.SaveAssignments("env_id", "other",
		map[string]*decision.VisitorCache{"vg": {VariationID: "v"}}, time.Now()))

	assert.Equal(t, 1, len(manager.current), "the generation was not retired on age")
	assert.Equal(t, 1, len(manager.previous))
}
