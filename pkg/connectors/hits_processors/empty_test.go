package hits_processors

import (
	"testing"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/stretchr/testify/assert"
)

func TestEmptyTrackHits(t *testing.T) {
	processor := &EmptyHitProcessor{}
	err := processor.TrackHits(connectors.TrackingHits{})
	assert.Nil(t, err)
}
