package handlers

// modificationResponse represents a decision campaign variation modification
// nolint
type modificationResponse struct {
	Type  string                 `json:"type"`
	Value map[string]interface{} `json:"value"`
}

// variationResponse represents a decision campaign variation
// nolint
type variationResponse struct {
	ID            string               `json:"id"`
	Modifications modificationResponse `json:"modifications"`
	Reference     bool                 `json:"reference"`
}

// campaignResponse represents a decision campaign
// nolint
type campaignResponse struct {
	ID               string            `json:"id"`
	CustomID         string            `json:"-"`
	VariationGroupID string            `json:"variationGroupId"`
	Variation        variationResponse `json:"variation"`
}

//nolint
type campaignsBodyContextSwagger struct {
	KeyString string  `json:"key_string"`
	KeyNumber float64 `json:"key_number"`
	KeyBool   bool    `json:"key_bool"`
}

//nolint
type campaignsBodySwagger struct {
	VisitorID   string                      `json:"visitor_id" binding:"required"`
	AnonymousID *string                     `json:"anonymous_id"`
	Context     campaignsBodyContextSwagger `json:"context"`
	TriggerHit  bool                        `json:"trigger_hit"`
}

// nolint
type campaignsBody struct {
	VisitorID   string                 `json:"visitor_id" binding:"required"`
	AnonymousID *string                `json:"anonymous_id"`
	Context     map[string]interface{} `json:"context"`
	TriggerHit  *bool                  `json:"trigger_hit"`
}

// campaignsResponse represents the campaigns call response
// nolint
type campaignsResponse struct {
	VisitorID string             `json:"visitor_id"`
	Panic     bool               `json:"panic"`
	Campaigns []campaignResponse `json:"campaigns"`
}

// nolint
type activateBody struct {
	VisitorID        string  `json:"vid" binding:"required"`
	AnonymousID      *string `json:"aid"`
	CampaignID       string  `json:"cid" binding:"required"`
	VariationGroupID string  `json:"caid" binding:"required"`
	VariationID      string  `json:"vaid" binding:"required"`
}

// nolint
type errorMessage struct {
	Message string `json:"message"`
}
