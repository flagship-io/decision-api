package main

import (
	"crypto/tls"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/utils/config"
)

func getAssignmentsManager(cfg *config.Config) (assignmentsManager connectors.AssignmentsManager, err error) {
	switch cfg.GetStringDefault("cache.type", "") {
	case "memory":
		assignmentsManager = assignments_managers.InitMemoryManager()
	case "local":
		assignmentsManager, err = assignments_managers.InitLocalCacheManager(assignments_managers.LocalOptions{
			DbPath: cfg.GetStringDefault("cache.options.dbpath", "cache_data"),
		})
	case "redis":
		var tlsConfig *tls.Config
		if cfg.GetBool("cache.options.redisTls") {
			tlsConfig = &tls.Config{}
		}
		assignmentsManager, err = assignments_managers.InitRedisManager(assignments_managers.RedisOptions{
			Host:      cfg.GetStringDefault("cache.options.redisHost", "localhost:6379"),
			Username:  cfg.GetStringDefault("cache.options.redisUsername", ""),
			Password:  cfg.GetStringDefault("cache.options.redisPassword", ""),
			Db:        cfg.GetIntDefault("cache.options.redisDb", 0),
			TTL:       cfg.GetDurationDefault("cache.options.redisTtl", 3*30*24*time.Hour),
			LogLevel:  config.LoggerLevel,
			TLSConfig: tlsConfig,
		})
	default:
		assignmentsManager = &assignments_managers.EmptyManager{}
	}

	return assignmentsManager, err
}
