package models

import (
	"fmt"
	"time"

	common "github.com/flagship-io/flagship-common"
)

type Environment struct {
	Common          *common.Environment
	HasIntegrations bool
}

type MappableHit interface {
	ToMap() map[string]interface{}
	ComputeQueueTime()
}

// CampaignActivation represents a single campaign activation
type CampaignActivation struct {
	EnvID           string `json:"cid"`
	VisitorID       string `json:"vid"`
	CustomerID      string `json:"cuid"`
	CampaignID      string `json:"caid"`
	VariationID     string `json:"vaid"`
	Timestamp       int64
	PersistActivate bool
	QA              bool  `json:"qa"`
	QueueTime       int64 `json:"qt"`
}

func (c *CampaignActivation) ComputeQueueTime() {
	c.QueueTime += time.Now().UnixMilli() - c.Timestamp
}

func (c *CampaignActivation) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"cid":  c.EnvID,
		"vid":  c.VisitorID,
		"caid": c.CampaignID,
		"vaid": c.VariationID,
		"qt":   c.QueueTime,
		"qa":   c.QA,
		"t":    "CAMPAIGN",
	}

	if c.CustomerID != "" {
		result["cuid"] = c.CustomerID
	}

	return result
}

type VisitorContext struct {
	EnvID      string                 `json:"cid"`
	VisitorID  string                 `json:"vid"`
	CustomerID string                 `json:"cuid"`
	Context    map[string]interface{} `json:"s"`
	Partner    string                 `json:"pa,omitempty"`
	Timestamp  int64
	QueueTime  int64 `json:"qt"`
}

func (c *VisitorContext) ComputeQueueTime() {
	c.QueueTime = time.Now().UnixMilli() - c.Timestamp
}

func (c *VisitorContext) ToMap() map[string]interface{} {
	segmentsString := map[string]string{}
	for k, v := range c.Context {
		segmentsString[k] = fmt.Sprintf("%v", v)
	}
	result := map[string]interface{}{
		"cid": c.EnvID,
		"vid": c.VisitorID,
		"s":   segmentsString,
		"qt":  c.QueueTime,
		"t":   "SEGMENT",
	}

	if c.CustomerID != "" {
		result["cuid"] = c.CustomerID
	}

	return result
}
