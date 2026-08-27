package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	*viper.Viper
}

func NewFromFilename(name string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(name)

	v.SetDefault("address", ServerAddress)
	v.SetDefault("cors.enabled", ServerCorsEnabled)
	v.SetDefault("cors.allowed_origins", ServerCorsAllowedOrigins)
	v.SetDefault("cors.allowed_headers", ServerCorsAllowedHeaders)
	v.SetDefault("log.level", LoggerLevel)
	v.SetDefault("log.format", LoggerFormat)
	v.SetDefault("polling_interval", CDNLoaderPollingInterval)
	v.SetDefault("cache.options.redisHost", RedisAddr)

	// replace dot in key name by underscore
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return &Config{v}, fmt.Errorf("config file could not be read: %w. Fallback to environment variables", err)
	}

	return &Config{v}, nil
}

func (c *Config) GetStringDefault(key, def string) string {
	if !c.IsSet(key) {
		return def
	}

	return c.GetString(key)
}

func (c *Config) GetIntDefault(key string, def int) int {
	if !c.IsSet(key) {
		return def
	}

	return c.GetInt(key)
}

func (c *Config) GetDurationDefault(key string, def time.Duration) time.Duration {
	if !c.IsSet(key) {
		return def
	}

	return c.GetDuration(key)
}
