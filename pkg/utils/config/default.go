package config

import "time"

const (
	ServerAddress = ":8080"
	LoggerLevel   = "warning"

	CDNLoaderPollingInterval = time.Minute * 1

	AddonCacheRedisAddr         = "localhost:6379"
	AddonCacheRedisPrefixKey    = "visitor-info:"
	AddonCacheRedisPoolSize     = 100
	AddonCacheRedisMaxRetries   = 2
	AddonCacheRedisTimeoutRead  = time.Second * 1
	AddonCacheRedisTimeoutWrite = time.Second * 1
)
