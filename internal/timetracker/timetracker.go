package timetracker

import (
	"os"
	"time"

	common "github.com/flagship-io/flagship-common"
)

// NewTracker Creates a tracker
func NewTracker() *common.Tracker {
	return &common.Tracker{
		StartTime: time.Now(),
		Enabled:   os.Getenv("TRACK_PERFORMANCE") == "true",
	}
}
