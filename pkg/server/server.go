package server

import (
	"errors"
	"net/http"

	_ "github.com/flagship-io/decision-api/docs"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/handlers"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	httpSwagger "github.com/swaggo/http-swagger"
)

type ServerOptions struct {
	experienceTracker  connectors.HitsProcessor
	environmentLoader  connectors.EnvironmentLoader
	assignmentsManager connectors.AssignmentsManager
	logger             *logger.Logger
}

type ServerOptionsBuilder func(*ServerOptions)

func WithExperienceTracker(tracker connectors.HitsProcessor) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.experienceTracker = tracker
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

type Server struct {
	options    *ServerOptions
	httpServer *http.ServeMux
}

func (srv *Server) Listen(addr string) error {
	return http.ListenAndServe(addr, srv.httpServer)
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
func CreateServer(envID string, apiKey string, opts ...ServerOptionsBuilder) (*Server, error) {
	serverOptions := &ServerOptions{}

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

	if serverOptions.experienceTracker == nil {
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
			HitsProcessor:      serverOptions.experienceTracker,
			EnvironmentLoader:  serverOptions.environmentLoader,
			AssignmentsManager: serverOptions.assignmentsManager,
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/v2/campaigns", handlers.Campaigns(context))
	mux.HandleFunc("/v2/campaigns/*", handlers.Campaign(context))
	mux.HandleFunc("/v2/activate", handlers.Activate(context))
	mux.HandleFunc("/v2/activate-batch", handlers.ActivateMultiple(context))
	mux.HandleFunc("/v2/flags", handlers.Flags(context))
	mux.HandleFunc("/v2/swagger/", httpSwagger.WrapHandler)

	server := &Server{
		options:    serverOptions,
		httpServer: mux,
	}

	return server, nil
}
