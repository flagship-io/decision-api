package main

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/flagship-io/decision-api/pkg/config"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/stretchr/testify/assert"
)

func TestGetAssignmentsManager(t *testing.T) {
	cfg, _ := config.NewFromFilename("test")

	cfg.Set("cache.type", "")
	assignmentsManager, err := getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, assignmentsManager, &assignments_managers.EmptyManager{})

	cfg.Set("cache.type", "local")
	assignmentsManager, err = getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, &assignments_managers.LocalManager{}, assignmentsManager)

	cfg.Set("cache.type", "memory")
	assignmentsManager, err = getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, &assignments_managers.MemoryManager{}, assignmentsManager)

	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()
	cfg.Set("cache.type", "redis")
	cfg.Set("cache.options.redisHost", s.Addr())
	assignmentsManager, err = getAssignmentsManager(cfg)
	assert.IsType(t, &assignments_managers.RedisManager{}, assignmentsManager)
	assert.Nil(t, err)

	cfg.Set("cache.type", "dynamo")
	assignmentsManager, err = getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, &assignments_managers.DynamoManager{}, assignmentsManager)
}
