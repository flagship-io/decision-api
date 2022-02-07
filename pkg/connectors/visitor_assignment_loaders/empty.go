package visitor_assignment_loaders

import (
	common "github.com/flagship-io/flagship-common"
)

type Empty struct{}

func (*Empty) LoadAssignments(envID string, visitorID string) (*common.VisitorAssignments, error) {
	return nil, nil
}
