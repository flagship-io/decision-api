package hits_processors

import (
	"context"

	"github.com/flagship-io/decision-api/pkg/connectors"
)

type MockHitProcessor struct {
	TrackedHits connectors.TrackingHits
}

func (d *MockHitProcessor) TrackHits(hits connectors.TrackingHits) error {
	d.TrackedHits = hits
	return nil
}

func (d *MockHitProcessor) Shutdown(context.Context) error {
	return nil
}
