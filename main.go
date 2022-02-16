package main

import (
	"flag"
	"log"

	"github.com/flagship-io/decision-api/cmd/server"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/experience_trackers"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

func main() {
	filename := flag.String("config", "config.yaml", "Path the configuration file")
	flag.Parse()

	cfg, err := config.NewFromFilename(*filename)
	if err != nil {
		log.Printf("config loaded with error: %v", err)
	}

	lvl := cfg.GetStringDefault("log_level", config.LoggerLevel)
	log := logger.New(lvl, "server")

	var assignmentManager connectors.AssignmentsManager
	err = nil
	switch cfg.GetStringDefault("cache_type", "") {
	case "memory":
		assignmentManager = &assignments_managers.InMemory{}
	case "local":
		assignmentManager, err = assignments_managers.InitLocalCacheManager(assignments_managers.LocalOptions{
			DbPath: cfg.GetStringDefault("cache_options_dbpath", ""),
		})
	case "redis":
		assignmentManager, err = assignments_managers.InitRedisManager(assignments_managers.RedisOptions{
			Host:     cfg.GetStringDefault("cache_options_redishost", "localhost:6379"),
			Username: cfg.GetStringDefault("cache_options_redisusername", ""),
			Password: cfg.GetStringDefault("cache_options_redispassword", ""),
			Db:       cfg.GetIntDefault("cache_options_redisdb", 0),
			Logger:   log,
		})
	default:
		assignmentManager = &assignments_managers.Empty{}
	}

	if err != nil {
		log.Fatalf("error occured when initializing assignment cache manager: %v", err)
	}

	server, err := server.CreateServer(
		cfg.GetString("ENV_ID"),
		cfg.GetString("API_KEY"),
		server.WithLogger(log),
		server.WithEnvironmentLoader(
			environment_loaders.NewCDNLoader(
				environment_loaders.WithLogLevel(lvl),
				environment_loaders.WithPollingInterval(cfg.GetDuration("polling_interval"))),
		),
		server.WithExperienceTracker(experience_trackers.NewDataCollectTracker(lvl)),
		server.WithAssignmentsManager(assignmentManager),
	)

	if err != nil {
		log.Fatalf("error when creating server: %v", err)
	}

	log.Info("server started on port 8080")
	server.Listen(":8080")
}
