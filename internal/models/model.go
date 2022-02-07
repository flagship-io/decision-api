package models

import "time"

// CampaignActivation represents a single campaign activation
type CampaignActivation struct {
	EnvID       string `json:"cid"`
	VisitorID   string `json:"vid"`
	CustomerID  string `json:"cuid"`
	CampaignID  string `json:"caid"`
	VariationID string `json:"vaid"`
	Timestamp   int64
	QueueTime   int64 `json:"qt"`
}

func (c *CampaignActivation) ComputeQueueTime() {
	c.QueueTime = time.Now().UnixMilli() - c.Timestamp
}

func (c *CampaignActivation) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"cid":  c.EnvID,
		"vid":  c.VisitorID,
		"caid": c.CampaignID,
		"vaid": c.VariationID,
		"qt":   c.QueueTime,
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
	Timestamp  int64
	QueueTime  int64 `json:"qt"`
}

func (c *VisitorContext) ComputeQueueTime() {
	c.QueueTime = time.Now().UnixMilli() - c.Timestamp
}

func (c *VisitorContext) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"cid": c.EnvID,
		"vid": c.VisitorID,
		"s":   c.Context,
		"qt":  c.QueueTime,
	}

	if c.CustomerID != "" {
		result["cuid"] = c.CustomerID
	}

	return result
}
