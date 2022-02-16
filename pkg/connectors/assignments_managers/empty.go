package assignments_managers

import (
	"time"

	common "github.com/flagship-io/flagship-common"
)

type Empty struct{}

func (*Empty) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	return nil, nil
}

func (*Empty) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) error {
	return nil
}
