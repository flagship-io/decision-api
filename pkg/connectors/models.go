package connectors

import (
	"time"

	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"
)

type DecisionContext struct {
	EnvID  string
	APIKey string
	Logger *logger.Logger
	Connectors
}

type Connectors struct {
	HitProcessor       HitProcessor
	EnvironmentLoader  EnvironmentLoader
	AssignmentsManager AssignmentsManager
}

type TrackingHits struct {
	CampaignActivations []*models.CampaignActivation
	VisitorContext      []*models.VisitorContext
}

type HitProcessor interface {
	TrackHits(hits TrackingHits) error
}

type EnvironmentLoader interface {
	Init(envID string, APIKey string) error
	LoadEnvironment(envID string, APIKey string) (*common.Environment, error)
}

type AssignmentsManager interface {
	LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error)
	SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error
}
