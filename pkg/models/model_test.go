package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCampaignActivationComputeQueueTime(t *testing.T) {
	c := CampaignActivation{
		Timestamp: time.Now().UnixMilli() - 100,
	}
	c.ComputeQueueTime()
	assert.Equal(t, int64(100), c.QueueTime)

	c = CampaignActivation{
		Timestamp: time.Now().UnixMilli() - 100,
		QueueTime: 50,
	}
	c.ComputeQueueTime()
	assert.Equal(t, int64(150), c.QueueTime)
}

func TestCampaignActivationToMap(t *testing.T) {
	c := CampaignActivation{
		EnvID:       "env_id",
		VisitorID:   "vid",
		CustomerID:  "cuid",
		CampaignID:  "caid",
		VariationID: "vaid",
		QueueTime:   100,
	}
	obj := c.ToMap()
	assert.EqualValues(t, map[string]interface{}{
		"cid":  "env_id",
		"vid":  "vid",
		"cuid": "cuid",
		"caid": "caid",
		"vaid": "vaid",
		"qt":   int64(100),
		"t":    "CAMPAIGN",
	}, obj)
}

func TestVisitorContextComputeQueueTime(t *testing.T) {
	c := VisitorContext{
		Timestamp: time.Now().UnixMilli() - 100,
	}
	c.ComputeQueueTime()
	assert.Equal(t, int64(100), c.QueueTime)
}

func TestVisitorContextToMap(t *testing.T) {
	c := VisitorContext{
		EnvID:      "env_id",
		VisitorID:  "vid",
		CustomerID: "cuid",
		Context: map[string]interface{}{
			"k": "v",
		},
		QueueTime: 100,
	}
	obj := c.ToMap()
	assert.EqualValues(t, map[string]interface{}{
		"cid":  "env_id",
		"vid":  "vid",
		"cuid": "cuid",
		"s": map[string]string{
			"k": "v",
		},
		"qt": int64(100),
		"t":  "SEGMENT",
	}, obj)
}
