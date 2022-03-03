package assignments_managers

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/flagship-io/decision-api/pkg/connectors"
	decision "github.com/flagship-io/flagship-common"
	"github.com/stretchr/testify/assert"
)

func TestRedisCache(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	// Wrong host test
	_, err = InitRedisManager(RedisOptions{
		Host: "localhost:4567",
	})
	assert.NotEqual(t, nil, err)

	envID := "env_id"
	visID := "visID"
	notInitialized := &RedisManager{}
	_, err = notInitialized.LoadAssignments(envID, visID)
	assert.Equal(t, "redis cache manager not initialized", err.Error())

	err = notInitialized.SaveAssignments(envID, visID, nil, time.Now(), connectors.SaveAssignmentsContext{})
	assert.Equal(t, "redis cache manager not initialized", err.Error())

	m, err := InitRedisManager(RedisOptions{
		Host: s.Addr(),
	})

	assert.Equal(t, nil, err)

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
