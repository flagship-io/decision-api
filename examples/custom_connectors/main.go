package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/server"
	common "github.com/flagship-io/flagship-common"
)

type CustomAssignmentManager struct {
}

func (m *CustomAssignmentManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	// TODO implement this method
	return nil, nil
}

func (m *CustomAssignmentManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time, context connectors.SaveAssignmentsContext) error {
	// TODO implement this method
	return nil
}

func main() {
	srv, err := server.CreateServer(
		os.Getenv("ENV_ID"),
		os.Getenv("API_KEY"),
		":8080",
		server.WithAssignmentsManager(&CustomAssignmentManager{}),
	)

	if err != nil {
		log.Fatalf("error when creating server: %v", err)
	}

	log.Printf("server listening on :8080")
	if err := srv.Listen(); err != http.ErrServerClosed {
		log.Fatalf("error when starting server: %v", err)
	}
}
