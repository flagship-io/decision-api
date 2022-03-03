package hits_processors

import (
	"github.com/flagship-io/decision-api/pkg/connectors"
)

type EmptyHitProcessor struct {
}

func (d *EmptyHitProcessor) TrackHits(hits connectors.TrackingHits) error {
	return nil
}
