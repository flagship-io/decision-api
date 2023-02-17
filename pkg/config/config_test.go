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

func TestGetDefaultString(t *testing.T) {
	cfg, _ := NewFromFilename("")
	addr := cfg.GetDefaultString("address", "default")
	assert.Equal(t, ServerAddress, addr)
	addr = cfg.GetDefaultString("not_exists", "default")
	assert.Equal(t, "default", addr)
}

func TestGetDefaultInt(t *testing.T) {
	cfg, _ := NewFromFilename("")
	cfg.Set("test", 1)
	val := cfg.GetDefaultInt("test", 2)
	assert.Equal(t, 1, val)
	val = cfg.GetDefaultInt("not_exists", 2)
	assert.Equal(t, 2, val)
}

func TestGetDefaultDuration(t *testing.T) {
	cfg, _ := NewFromFilename("")
	cfg.Set("test", 1*time.Second)
	val := cfg.GetDefaultDuration("test", 2*time.Minute)
	assert.Equal(t, 1*time.Second, val)
	val = cfg.GetDefaultDuration("not_exists", 2*time.Minute)
	assert.Equal(t, 2*time.Minute, val)
}
