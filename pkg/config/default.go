package config

import "time"

const (
	ServerAddress            = ":8080"
	ServerCorsEnabled        = true
	ServerCorsAllowedOrigins = "*"
	ServerCorsAllowedHeaders = "Content-Type,Authorization,X-Api-Key,X-Sdk-Client,X-Sdk-Version,X-Pop"
	LoggerLevel              = "warning"
	LoggerFormat             = "text"

	CDNLoaderPollingInterval = time.Minute * 1

	RedisAddr = "localhost:6379"
)
