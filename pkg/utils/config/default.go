package config

import "time"

const (
	ServerAddress = ":8080"
	LoggerLevel   = "warning"

	CDNLoaderPollingInterval = time.Minute * 1

	AddonCacheRedisAddr = "localhost:6379"
)
