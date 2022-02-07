package server

import (
	"errors"
	"net/http"

	"github.com/flagship-io/decision-api/internal/models"
	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/handlers"
)

type ServerOptions struct {
	experienceTracker       connectors.HitProcessor
	environmentLoader       connectors.EnvironmentLoader
	visitorAssignmentLoader connectors.VisitorAssignmentLoader
	visitorAssignmentSaver  connectors.VisitorAssignmentSaver
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

func WithVisitorAssignmentLoader(loader connectors.VisitorAssignmentLoader) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.visitorAssignmentLoader = loader
	}
}

func WithVisitorAssignmentSaver(saver connectors.VisitorAssignmentSaver) ServerOptionsBuilder {
	return func(h *ServerOptions) {
		h.visitorAssignmentSaver = saver
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

	if serverOptions.environmentLoader == nil {
		return nil, errors.New("missing mandatory environmentLoader connector")
	}

	if serverOptions.experienceTracker == nil {
		return nil, errors.New("missing mandatory experienceTracker connector")
	}

	if serverOptions.visitorAssignmentLoader == nil {
		return nil, errors.New("missing mandatory visitorAssignmentLoader connector")
	}

	if serverOptions.visitorAssignmentSaver == nil {
		return nil, errors.New("missing mandatory visitorAssignmentSaver connector")
	}

	context := &models.DecisionContext{
		APIKey: apiKey,
		EnvID:  envID,
		Connectors: connectors.Connectors{
			HitProcessor:            serverOptions.experienceTracker,
			EnvironmentLoader:       serverOptions.environmentLoader,
			VisitorAssignmentLoader: serverOptions.visitorAssignmentLoader,
			VisitorAssignmentSaver:  serverOptions.visitorAssignmentSaver,
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
