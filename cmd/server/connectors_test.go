package main

import (
	"testing"

	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/stretchr/testify/assert"
)

func TestGetAssignmentsManager(t *testing.T) {
	cfg, _ := config.NewFromFilename("test")

	cfg.Set("cache.type", "")
	assignmentsManager, err := getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, assignmentsManager, &assignments_managers.EmptyManager{})

	cfg.Set("cache.type", "redis")
	_, err = getAssignmentsManager(cfg)
	assert.NotNil(t, err)

	cfg.Set("cache.type", "local")
	assignmentsManager, err = getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, &assignments_managers.LocalManager{}, assignmentsManager)

	cfg.Set("cache.type", "memory")
	assignmentsManager, err = getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, &assignments_managers.MemoryManager{}, assignmentsManager)
}
