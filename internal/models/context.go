package models

import "github.com/flagship-io/decision-api/pkg/connectors"

type DecisionContext struct {
	EnvID  string
	APIKey string
	connectors.Connectors
}
