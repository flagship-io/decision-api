package config

import "time"

const (
	ServerAddress            = ":8080"
	ServerCorsEnabled        = true
	ServerCorsAllowedOrigins = "*"
	LoggerLevel              = "warning"
	LoggerFormat             = "text"

	CDNLoaderPollingInterval = time.Minute * 1

	RedisAddr = "localhost:6379"
)
