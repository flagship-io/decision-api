package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/server"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	"github.com/sirupsen/logrus"
)

var shutdownTimeout = 3 * time.Second

func createLogger(cfg *config.Config) *logger.Logger {
	lvl := cfg.GetStringDefault("log.level", config.LoggerLevel)
	format := cfg.GetStringDefault("log.format", config.LoggerFormat)
	log := logger.New(lvl, "server")

	if format == "json" {
		log.Logger.SetFormatter(&logrus.JSONFormatter{})
	}
	return log
}

func createServer(cfg *config.Config, log *logger.Logger) (*server.Server, error) {
	logLvl := log.Logger.Level.String()
	log.Info("initializing assignment cache manager from configuration")
	assignmentManager, err := getAssignmentsManager(cfg)
	if err != nil {
		log.Fatalf("error occurred when initializing assignment cache manager: %v", err)
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
		server.WithHitsProcessor(hits_processors.NewDataCollectProcessor(hits_processors.WithLogLevel(logLvl))),
		server.WithAssignmentsManager(assignmentManager),
		server.WithCorsOptions(&models.CorsOptions{
			Enabled:        cfg.GetBool("cors.enabled"),
			AllowedOrigins: cfg.GetStringDefault("cors.allowed_origins", config.ServerCorsAllowedOrigins),
		}),
	)
}

func main() {
	cfgFilename := flag.String("config", "config.yaml", "Path the configuration file")
	flag.Parse()

	cfg, errCfg := config.NewFromFilename(*cfgFilename)
	logger := createLogger(cfg)

	if errCfg != nil {
		logger.Warn(errCfg)
	}

	srv, err := createServer(cfg, logger)
	if err != nil {
		logger.Fatalf("error when creating server: %v", err)
	}

	// Run server
	go func() {
		logger.Infof("Flagship Decision API server [%s] listening on %s", models.Version, cfg.GetStringDefault("address", ":8080"))
		if err := srv.Listen(); err != http.ErrServerClosed {
			logger.Fatalf("error when starting server: %v", err)
		}
	}()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	<-signalChannel

	// Try to gracefully shutdown the server
	ctx, cancelFunc := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelFunc()
	srv.Shutdown(ctx)
}
