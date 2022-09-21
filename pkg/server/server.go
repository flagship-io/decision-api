package server

import (
	"context"
	"errors"
	"expvar"
	"net/http"
	"time"

	"github.com/flagship-io/decision-api/docs"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/handlers"
	"github.com/flagship-io/decision-api/pkg/handlers/middlewares"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/config"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
	httpSwagger "github.com/swaggo/http-swagger"
)

type ServerOptions struct {
	hitsProcessor      connectors.HitsProcessor
	environmentLoader  connectors.EnvironmentLoader
	assignmentsManager connectors.AssignmentsManager
	logger             *logger.Logger
	corsOptions        *models.CorsOptions
	recover            bool
}

type ServerOptionsBuilder func(*ServerOptions)

func WithHitsProcessor(processor connectors.HitsProcessor) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.hitsProcessor = processor
	}
}

func WithEnvironmentLoader(loader connectors.EnvironmentLoader) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.environmentLoader = loader
	}
}

func WithAssignmentsManager(manager connectors.AssignmentsManager) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.assignmentsManager = manager
	}
}

func WithLogger(logger *logger.Logger) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.logger = logger
	}
}

func WithCorsOptions(options *models.CorsOptions) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.corsOptions = options
	}
}

func WithRecover(enabled bool) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.recover = enabled
	}
}

type Server struct {
	options    *ServerOptions
	httpServer *http.Server
}

func (srv *Server) Listen() error {
	return srv.httpServer.ListenAndServe()
}

func wrapMiddlewares(serverOptions *ServerOptions, endpointName string, handler http.HandlerFunc) http.HandlerFunc {
	return middlewares.Recover(
		serverOptions.recover,
		middlewares.Metrics(endpointName,
			middlewares.Version(
				middlewares.Cors(serverOptions.corsOptions, handler))))
}

// @title Flagship Decision API
// @version 2.0
// @BasePath /v2
// @description This is the Flagship Decision API documentation

// @contact.name API Support
// @contact.url https://www.flagship.io
// @contact.email support@flagship.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
func CreateServer(envID string, apiKey string, addr string, opts ...ServerOptionsBuilder) (*Server, error) {

	// Dynamic swagger version
	docs.SwaggerInfo.Version = models.Version

	serverOptions := &ServerOptions{
		logger: logger.New(config.LoggerLevel, config.LoggerFormat, "server"),
		environmentLoader: environment_loaders.NewCDNLoader(
			environment_loaders.WithPollingInterval(config.CDNLoaderPollingInterval),
		),
		hitsProcessor:      hits_processors.NewDataCollectProcessor(),
		assignmentsManager: &assignments_managers.EmptyManager{},
		corsOptions: &models.CorsOptions{
			Enabled:        config.ServerCorsEnabled,
			AllowedOrigins: config.ServerCorsAllowedOrigins,
			AllowedHeaders: config.ServerCorsAllowedHeaders,
		},
		recover: true,
	}

	for _, opt := range opts {
		opt(serverOptions)
	}

	if envID == "" {
		return nil, errors.New("missing mandatory environment ID")
	}

	if apiKey == "" {
		return nil, errors.New("missing mandatory API Key")
	}

	if serverOptions.logger == nil {
		return nil, errors.New("missing mandatory logger")
	}

	if serverOptions.environmentLoader == nil {
		return nil, errors.New("missing mandatory environmentLoader connector")
	}

	if serverOptions.hitsProcessor == nil {
		return nil, errors.New("missing mandatory experienceTracker connector")
	}

	if serverOptions.assignmentsManager == nil {
		return nil, errors.New("missing mandatory visitorAssignmentLoader connector")
	}

	err := serverOptions.environmentLoader.Init(envID, apiKey)
	if err != nil {
		serverOptions.logger.Errorf("error when initializing environment loader: %v", err)
	}

	context := &connectors.DecisionContext{
		APIKey: apiKey,
		EnvID:  envID,
		Logger: serverOptions.logger,
		Connectors: connectors.Connectors{
			HitsProcessor:      serverOptions.hitsProcessor,
			EnvironmentLoader:  serverOptions.environmentLoader,
			AssignmentsManager: serverOptions.assignmentsManager,
		},
	}

	// set the logger for common package
	commonLogger := logger.New(serverOptions.logger.Level.String(), config.LoggerFormat, "common")
	common.SetLogger(&common.DefaultLogger{
		Entry: commonLogger.Entry,
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/v2/campaigns", wrapMiddlewares(serverOptions, "campaigns", handlers.Campaigns(context)))
	mux.HandleFunc("/v2/campaigns/", wrapMiddlewares(serverOptions, "campaign", handlers.Campaign(context)))
	mux.HandleFunc("/v2/activate", wrapMiddlewares(serverOptions, "activate", handlers.Activate(context)))
	mux.HandleFunc("/v2/flags", wrapMiddlewares(serverOptions, "flags", handlers.Flags(context)))
	mux.HandleFunc("/v2/metrics", wrapMiddlewares(serverOptions, "metrics", expvar.Handler().ServeHTTP))
	mux.HandleFunc("/v2/swagger/", httpSwagger.WrapHandler)

	server := &Server{
		options: serverOptions,
		httpServer: &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			Addr:         addr,
			Handler:      middlewares.RequestLogger(serverOptions.logger, mux),
		}}

	return server, nil
}

func (s *Server) Shutdown(ctx context.Context) {
	s.options.logger.Info("shutting server down")
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.options.logger.Errorf("error when shutting server down: %v", err)
	}

	s.options.logger.Info("cleaning remaining hits")
	err = s.options.hitsProcessor.Shutdown(ctx)
	if err != nil {
		s.options.logger.Errorf("error when shutting server down: %v", err)
	}
}
