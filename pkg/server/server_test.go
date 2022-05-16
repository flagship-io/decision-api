package server

import (
	"testing"

	_ "github.com/flagship-io/decision-api/docs"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	"github.com/stretchr/testify/assert"
)

func TestCreateServer(t *testing.T) {
	envID := ""
	apiKey := ""
	_, err := CreateServer(envID, apiKey, ":8080")
	assert.NotNil(t, err)

	envID = "env_id"
	_, err = CreateServer(envID, apiKey, ":8080")
	assert.NotNil(t, err)

	apiKey = "api_key"
	server, err := CreateServer(envID, apiKey, ":8080")
	assert.Nil(t, err)
	assert.Equal(t, config.ServerCorsEnabled, server.options.corsOptions.Enabled)
	assert.Equal(t, config.ServerCorsAllowedOrigins, server.options.corsOptions.AllowedOrigins)
	assert.Equal(t, config.ServerCorsAllowedHeaders, server.options.corsOptions.AllowedHeaders)
	assert.Equal(t, config.LoggerLevel, server.options.logger.Logger.Level.String())

	_, err = CreateServer(envID, apiKey, ":8080", WithAssignmentsManager(nil))
	assert.NotNil(t, err)

	_, err = CreateServer(envID, apiKey, ":8080", WithEnvironmentLoader(nil))
	assert.NotNil(t, err)

	_, err = CreateServer(envID, apiKey, ":8080", WithHitsProcessor(nil))
	assert.NotNil(t, err)

	_, err = CreateServer(envID, apiKey, ":8080", WithLogger(nil))
	assert.NotNil(t, err)

	assignmentManager := assignments_managers.InitMemoryManager()
	hitsProcessor := &hits_processors.MockHitProcessor{}
	environmentLoader := &environment_loaders.MockLoader{}
	log := logger.New("debug", "test")
	server, err = CreateServer(
		envID,
		apiKey,
		":8080",
		WithAssignmentsManager(assignmentManager),
		WithHitsProcessor(hitsProcessor),
		WithEnvironmentLoader(environmentLoader),
		WithLogger(log))
	assert.Nil(t, err)
	assert.Equal(t, assignmentManager, server.options.assignmentsManager)
	assert.Equal(t, hitsProcessor, server.options.hitsProcessor)
	assert.Equal(t, environmentLoader, server.options.environmentLoader)
	assert.Equal(t, log, server.options.logger)
}
