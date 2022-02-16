package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const configPath = "../../../test/config-test.yaml"

func TestConfig_GetStringDefault(t *testing.T) {
	cfg, err := NewFromFilename(configPath)
	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	_ = os.Setenv("MY_KEY", "my-value")

	tt := []struct {
		key           string
		def, expected string
	}{
		{key: "logger.level", def: "warning", expected: "fatal"},
		{key: "not.found", def: "default", expected: "default"},
		{key: "my.key", def: "default", expected: "my-value"},
	}

	for i := range tt {
		out := cfg.GetStringDefault(tt[i].key, tt[i].def)
		assert.Equal(t, tt[i].expected, out)
	}
}

func TestConfig_GetDurationDefault(t *testing.T) {
	cfg, err := NewFromFilename(configPath)
	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	_ = os.Setenv("MY_DURATION", "60s")

	tt := []struct {
		key           string
		def, expected time.Duration
	}{
		{key: "server.timeout.read", def: time.Second, expected: time.Second * 10},
		{key: "not.found", def: time.Second, expected: time.Second},
		{key: "my.duration", def: time.Second, expected: time.Minute},
	}

	for i := range tt {
		out := cfg.GetDurationDefault(tt[i].key, tt[i].def)
		assert.Equal(t, tt[i].expected, out)
	}
}

func TestConfig_GetIntDefault(t *testing.T) {
	cfg, err := NewFromFilename(configPath)
	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	_ = os.Setenv("MY_INT", "8")

	tt := []struct {
		key           string
		def, expected int
	}{
		{key: "addon.cache.redis.max_retries", def: 12, expected: 4},
		{key: "not.found", def: 28, expected: 28},
		{key: "my.int", def: 7, expected: 8},
	}

	for i := range tt {
		out := cfg.GetIntDefault(tt[i].key, tt[i].def)
		assert.Equal(t, tt[i].expected, out)
	}
}
