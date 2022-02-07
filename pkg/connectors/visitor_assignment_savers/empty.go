package visitor_assignment_savers

import (
	"time"

	common "github.com/flagship-io/flagship-common"
)

type Empty struct{}

func (*Empty) SaveAssignments(envID string, visitorID string, vgIDAssignments map[string]*common.VisitorCache, date time.Time) {

}
