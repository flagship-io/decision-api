package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flagship-io/decision-api/pkg/config"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/logger"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/server"
)

var shutdownTimeout = 3 * time.Second

func createLogger(cfg *config.Config) *logger.Logger {
	lvl := cfg.GetDefaultString("log.level", config.LoggerLevel)
	format := cfg.GetDefaultString("log.format", config.LoggerFormat)

	return logger.New(lvl, logger.LogFormat(format), "Server")
}

func createServer(cfg *config.Config, log *logger.Logger) (*server.Server, error) {
	logLvl := cfg.GetDefaultString("log.level", config.LoggerLevel)
	logFmt := cfg.GetDefaultString("log.format", config.LoggerFormat)

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
				environment_loaders.WithLogger(logLvl, logger.LogFormat(logFmt)),
				environment_loaders.WithPollingInterval(cfg.GetDuration("polling_interval"))),
		),
		server.WithHitsProcessor(hits_processors.NewDataCollectProcessor(hits_processors.WithLogger(logLvl, logger.LogFormat(logFmt)))),
		server.WithAssignmentsManager(assignmentManager),
		server.WithCorsOptions(&models.CorsOptions{
			Enabled:        cfg.GetBool("cors.enabled"),
			AllowedOrigins: cfg.GetDefaultString("cors.allowed_origins", config.ServerCorsAllowedOrigins),
			AllowedHeaders: cfg.GetDefaultString("cors.allowed_headers", config.ServerCorsAllowedHeaders),
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
		logger.Infof("Flagship Decision API server [%s] listening on %s", models.Version, cfg.GetDefaultString("address", ":8080"))
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
