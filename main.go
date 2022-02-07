package main

import (
	"log"
	"os"

	"github.com/flagship-io/decision-api/cmd/server"
	"github.com/flagship-io/decision-api/pkg/connectors/environment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/experience_trackers"
	"github.com/flagship-io/decision-api/pkg/connectors/visitor_assignment_loaders"
	"github.com/flagship-io/decision-api/pkg/connectors/visitor_assignment_savers"
)

func main() {
	server, err := server.CreateServer(
		"env_id",
		"api_key",
		server.WithEnvironmentLoader(
			environment_loaders.NewCDNLoader(
				os.Getenv("ENV_ID"),
				os.Getenv("API_KEY"),
			),
		),
		server.WithExperienceTracker(experience_trackers.NewDataCollectTracker()),
		server.WithVisitorAssignmentLoader(&visitor_assignment_loaders.Empty{}),
		server.WithVisitorAssignmentSaver(&visitor_assignment_savers.Empty{}),
	)

	if err != nil {
		log.Fatalf("error when creating server: %v", err)
	}

	log.Println("server started on port 8080")
	server.Listen(":8080")
}
