package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFromFilename(t *testing.T) {
	cfg, err := NewFromFilename("")
	assert.NotNil(t, cfg)
	assert.NotNil(t, err)

	assert.Equal(t, cfg.GetString("address"), ServerAddress)
	assert.Equal(t, cfg.GetBool("cors.enabled"), ServerCorsEnabled)
	assert.Equal(t, cfg.GetString("cors.allowed_origins"), ServerCorsAllowedOrigins)
	assert.Equal(t, cfg.GetString("cors.allowed_headers"), ServerCorsAllowedHeaders)
	assert.Equal(t, cfg.GetString("log.level"), LoggerLevel)
	assert.Equal(t, cfg.GetString("log.format"), LoggerFormat)
	assert.Equal(t, cfg.GetDuration("polling_interval"), CDNLoaderPollingInterval)
	assert.Equal(t, cfg.GetString("cache.options.redisHost"), RedisAddr)
}

func TestGetStringDefault(t *testing.T) {
	cfg, _ := NewFromFilename("")
	addr := cfg.GetStringDefault("address", "default")
	assert.Equal(t, ServerAddress, addr)
	addr = cfg.GetStringDefault("not_exists", "default")
	assert.Equal(t, "default", addr)
}

func TestGetIntDefault(t *testing.T) {
	cfg, _ := NewFromFilename("")
	cfg.Set("test", 1)
	val := cfg.GetIntDefault("test", 2)
	assert.Equal(t, 1, val)
	val = cfg.GetIntDefault("not_exists", 2)
	assert.Equal(t, 2, val)
}

func TestGetDurationDefault(t *testing.T) {
	cfg, _ := NewFromFilename("")
	cfg.Set("test", 1*time.Second)
	val := cfg.GetDurationDefault("test", 2*time.Minute)
	assert.Equal(t, 1*time.Second, val)
	val = cfg.GetDurationDefault("not_exists", 2*time.Minute)
	assert.Equal(t, 2*time.Minute, val)
}

func TestGetBoolDefault(t *testing.T) {
	cfg, _ := NewFromFilename("")

	assert.True(t, cfg.GetBoolDefault("not_exists", true), "an unset key must keep the default")
	assert.False(t, cfg.GetBoolDefault("not_exists", false))

	// The case that matters for an option that is on by default: turning it off has to work, and
	// GetBool alone cannot tell "set to false" apart from "not set".
	cfg.Set("test", false)
	assert.False(t, cfg.GetBoolDefault("test", true), "an option defaulting to true could not be turned off")

	cfg.Set("test", true)
	assert.True(t, cfg.GetBoolDefault("test", false))
}

// TestGetBoolDefaultFromEnvironment covers how an operator actually turns an option off in a
// container: the environment variable, not the configuration file.
func TestGetBoolDefaultFromEnvironment(t *testing.T) {
	t.Setenv("HITS_DEDUPLICATE_CONTEXT", "false")
	cfg, _ := NewFromFilename("")

	assert.False(t, cfg.GetBoolDefault("hits.deduplicate_context", true),
		"HITS_DEDUPLICATE_CONTEXT=false must turn off an option that defaults to true")
}
