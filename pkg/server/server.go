package server

import (
	"context"
	"errors"
	"expvar"
	"net/http"
	"time"

	_ "github.com/flagship-io/decision-api/docs"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/connectors/assignments_managers"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/hits_processors"
	"github.com/flagship-io/decision-api/pkg/handlers"
	"github.com/flagship-io/decision-api/pkg/handlers/middlewares"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
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

func WithHitsProcessor(tracker connectors.HitsProcessor) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.hitsProcessor = tracker
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
	serverOptions := &ServerOptions{
		logger:             logger.New("error", "server"),
		environmentLoader:  environment_loaders.NewCDNLoader(),
		hitsProcessor:      hits_processors.NewDataCollectTracker("error"),
		assignmentsManager: &assignments_managers.EmptyManager{},
		recover:            true,
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

	mux := http.NewServeMux()

	mux.HandleFunc("/v2/campaigns", middlewares.Recover(serverOptions.recover, middlewares.Metrics("campaigns", middlewares.Cors(serverOptions.corsOptions, handlers.Campaigns(context)))))
	mux.HandleFunc("/v2/campaigns/*", middlewares.Recover(serverOptions.recover, middlewares.Metrics("campaign", middlewares.Cors(serverOptions.corsOptions, handlers.Campaign(context)))))
	mux.HandleFunc("/v2/activate", middlewares.Recover(serverOptions.recover, middlewares.Metrics("activate", middlewares.Cors(serverOptions.corsOptions, handlers.Activate(context)))))
	mux.HandleFunc("/v2/flags", middlewares.Recover(serverOptions.recover, middlewares.Metrics("flags", middlewares.Cors(serverOptions.corsOptions, handlers.Flags(context)))))
	mux.HandleFunc("/v2/swagger/", httpSwagger.WrapHandler)
	mux.HandleFunc("/v2/metrics", expvar.Handler().ServeHTTP)

	server := &Server{
		options: serverOptions,
		httpServer: &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			Addr:         addr,
			Handler:      mux,
		},
	}

	return server, nil
}

func (s *Server) Shutdown(context context.Context) error {
	return s.httpServer.Shutdown(context)
}
