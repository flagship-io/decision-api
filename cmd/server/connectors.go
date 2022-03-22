package main

import (
	"crypto/tls"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/utils/config"
)

func getAssignmentsManager(cfg *config.Config) (assignmentsManager connectors.AssignmentsManager, err error) {
	switch cfg.GetStringDefault("cache_type", "") {
	case "memory":
		assignmentsManager = assignments_managers.InitMemoryManager()
	case "local":
		assignmentsManager, err = assignments_managers.InitLocalCacheManager(assignments_managers.LocalOptions{
			DbPath: cfg.GetStringDefault("cache_options_dbpath", "cache_data"),
		})
	case "redis":
		var tlsConfig *tls.Config
		if cfg.GetBool("cache_options_redistls") {
			tlsConfig = &tls.Config{}
		}
		assignmentsManager, err = assignments_managers.InitRedisManager(assignments_managers.RedisOptions{
			Host:      cfg.GetStringDefault("cache_options_redishost", "localhost:6379"),
			Username:  cfg.GetStringDefault("cache_options_redisusername", ""),
			Password:  cfg.GetStringDefault("cache_options_redispassword", ""),
			Db:        cfg.GetIntDefault("cache_options_redisdb", 0),
			LogLevel:  config.LoggerLevel,
			TLSConfig: tlsConfig,
		})
	default:
		assignmentsManager = &assignments_managers.EmptyManager{}
	}

	return assignmentsManager, err
}
