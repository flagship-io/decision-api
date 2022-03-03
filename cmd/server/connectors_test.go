package main

import (
	"testing"

	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/stretchr/testify/assert"
)

func TestGetAssignmentsManager(t *testing.T) {
	cfg, _ := config.NewFromFilename("test")

	cfg.Set("cache_type", "")
	assignmentsManager, err := getAssignmentsManager(cfg)
	assert.Nil(t, err)
	assert.IsType(t, assignmentsManager, &assignments_managers.EmptyManager{})

	cfg.Set("cache_type", "redis")
	_, err = getAssignmentsManager(cfg)
	assert.NotNil(t, err)
}
