package assignments_managers

import (
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	common "github.com/flagship-io/flagship-common"
)

type EmptyManager struct{}

func (*EmptyManager) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	return nil, nil
}

func (d *EmptyManager) ShouldSaveAssignments(context connectors.SaveAssignmentsContext) bool {
	return true
}

func (*EmptyManager) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	return nil
}
