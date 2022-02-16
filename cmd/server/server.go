package server

import (
	"errors"
	"net/http"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/handlers"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

type ServerOptions struct {
	experienceTracker  connectors.HitProcessor
	environmentLoader  connectors.EnvironmentLoader
	AssignmentsManager connectors.AssignmentsManager
	logger             *logger.Logger
}

type ServerOptionsBuilder func(*ServerOptions)

func WithExperienceTracker(tracker connectors.HitProcessor) ServerOptionsBuilder {
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
		h.AssignmentsManager = manager
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

func CreateServer(envID string, apiKey string, opts ...ServerOptionsBuilder) (*Server, error) {
	serverOptions := &ServerOptions{}

	for _, opt := range opts {
		opt(serverOptions)
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

	if serverOptions.AssignmentsManager == nil {
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
			HitProcessor:       serverOptions.experienceTracker,
			EnvironmentLoader:  serverOptions.environmentLoader,
			AssignmentsManager: serverOptions.AssignmentsManager,
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/v2/campaigns", handlers.Campaigns(context))
	mux.HandleFunc("/v2/campaigns/*", handlers.Campaign(context))
	mux.HandleFunc("/v2/activate", handlers.Activate(context))
	mux.HandleFunc("/v2/activate-batch", handlers.ActivateMultiple(context))
	mux.HandleFunc("/v2/events", handlers.Events(context))
	mux.HandleFunc("/v2/flags", handlers.Flags(context))

	server := &Server{
		options:    serverOptions,
		httpServer: mux,
	}

	return server, nil
}
