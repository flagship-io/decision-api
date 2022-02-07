package connectors

import (
	"time"

	common "github.com/flagship-io/flagship-common"
)

type Connectors struct {
	HitProcessor            HitProcessor
	EnvironmentLoader       EnvironmentLoader
	VisitorAssignmentLoader VisitorAssignmentLoader
	VisitorAssignmentSaver  VisitorAssignmentSaver
}

type TrackingHit interface {
	ComputeQueueTime()
	ToMap() map[string]interface{}
}

type HitProcessor interface {
	TrackHits(hit []TrackingHit) error
}

type EnvironmentLoader interface {
	LoadEnvironment(envID string, APIKey string) (*common.Environment, error)
}

type VisitorAssignmentLoader interface {
	LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error)
}

type VisitorAssignmentSaver interface {
	SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time)
}
