package main

import (
	"flag"
	"log"

	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/server"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
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

	// set the logger for common package
	commonLogger := logger.New(lvl, "common")
	if err != nil {
		log.Warnf("could not parse log level %s", lvl)
	} else {
		common.SetLogger(&common.DefaultLogger{
			Entry: commonLogger.Entry,
		})
	}

	log.Infof("creating assignment cache manager from configuration")
	assignmentManager, err := getAssignmentsManager(cfg)
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
		server.WithHitsProcessor(hits_processors.NewDataCollectTracker(lvl)),
		server.WithAssignmentsManager(assignmentManager),
	)

	if err != nil {
		log.Fatalf("error when creating server: %v", err)
	}

	log.Infof("server listening on %s", cfg.GetStringDefault("address", ":8080"))
	err = server.Listen(cfg.GetStringDefault("address", ":8080"))

	if err != nil {
		log.Fatalf("error when starting server: %v", err)
	}
}
