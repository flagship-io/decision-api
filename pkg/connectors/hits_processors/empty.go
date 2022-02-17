package hits_processors

import (
	"github.com/flagship-io/decision-api/pkg/connectors"
)

type EmptyTracker struct {
}

func (d *EmptyTracker) TrackHits(hits connectors.TrackingHits) error {
	return nil
}
