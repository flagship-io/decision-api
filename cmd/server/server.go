package main

import (
	"flag"
	"log"
	"net/http"
	"sync"

	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/server"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
)

var srv *server.Server
var lock = &sync.Mutex{}

func createLogger(cfg *config.Config) *logger.Logger {
	lvl := cfg.GetStringDefault("log_level", config.LoggerLevel)
	return logger.New(lvl, "server")
}

func createServer(cfg *config.Config, log *logger.Logger) (*server.Server, error) {
	logLvl := log.Level.String()

	// set the logger for common package
	commonLogger := logger.New(logLvl, "common")
	common.SetLogger(&common.DefaultLogger{
		Entry: commonLogger.Entry,
	})

	log.Infof("creating assignment cache manager from configuration")
	assignmentManager, err := getAssignmentsManager(cfg)
	if err != nil {
		log.Fatalf("error occured when initializing assignment cache manager: %v", err)
	}

	return server.CreateServer(
		cfg.GetString("env_id"),
		cfg.GetString("api_key"),
		cfg.GetString("address"),
		server.WithLogger(log),
		server.WithEnvironmentLoader(
			environment_loaders.NewCDNLoader(
				environment_loaders.WithLogLevel(logLvl),
				environment_loaders.WithPollingInterval(cfg.GetDuration("polling_interval"))),
		),
		server.WithHitsProcessor(hits_processors.NewDataCollectTracker(logLvl)),
		server.WithAssignmentsManager(assignmentManager),
	)
}

func main() {
	filename := flag.String("config", "config.yaml", "Path the configuration file")
	flag.Parse()

	cfg, err := config.NewFromFilename(*filename)
	if err != nil {
		log.Printf("config loaded with error: %v", err)
	}

	log := createLogger(cfg)
	lock.Lock()
	srv, err = createServer(cfg, log)
	lock.Unlock()
	if err != nil {
		log.Fatalf("error when creating server: %v", err)
	}

	log.Infof("server listening on %s", cfg.GetStringDefault("address", ":8080"))
	err = srv.Listen()

	if err != http.ErrServerClosed {
		log.Fatalf("error when starting server: %v", err)
	}
}
