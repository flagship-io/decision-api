package handle

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/flagship-io/decision-api/pkg/utils/logger"
	common "github.com/flagship-io/flagship-common"

	"github.com/flagship-io/flagship-proto/decision_response"
	"gitlab.com/canarybay/protobuf/ptypes.git/decision_request"
)

// Request represents the infos of the requests needed for the decision API
type Request struct {
	DecisionRequest  *decision_request.DecisionRequest
	DecisionContext  *connectors.DecisionContext
	DecisionResponse *decision_response.DecisionResponse
	Environment      *common.Environment
	CampaignID       string
	Mode             string
	Extras           []string
	ExposeAllKeys    bool
	SendContextEvent bool
	Time             time.Time
	Logger           *logger.Logger
}

func NewRequestFromHTTP(req *http.Request) Request {
	campaign_id := strings.TrimPrefix(req.URL.Path, "/v2/campaigns/")
	if campaign_id == req.URL.Path {
		campaign_id = ""
	}
	return Request{
		Time:       time.Now(),
		CampaignID: campaign_id,
		Extras:     []string{},
	}
}

func (r Request) HasExtra(extra string) bool {
	for _, e := range r.Extras {
		if e == extra {
			return true
		}
	}
	return false
}

func shouldTriggerHit(request *decision_request.DecisionRequest) bool {
	if (request.GetTriggerHit() != nil && !request.GetTriggerHit().GetValue()) ||
		(request.GetActivate() != nil && !request.GetActivate().GetValue()) {
		return false
	}
	return true
}

// Decision returns a DecisionResponse from the DecisionRequest and the campaign information
func Decision(handleRequest *Request, tracker *common.Tracker) error {
	if handleRequest.Environment == nil {
		return errors.New("client context not initialized")
	}
	decisionResponse, err := common.GetDecision(
		common.Visitor{
			ID:            handleRequest.DecisionRequest.VisitorId.GetValue(),
			AnonymousID:   handleRequest.DecisionRequest.AnonymousId.GetValue(),
			DecisionGroup: handleRequest.DecisionRequest.DecisionGroup.GetValue(),
			Context:       handleRequest.DecisionRequest.Context,
		},
		*handleRequest.Environment,
		common.DecisionOptions{
			TriggerHit:        shouldTriggerHit(handleRequest.DecisionRequest),
			CampaignID:        handleRequest.CampaignID,
			Tracker:           tracker,
			IsCumulativeAlloc: true,
			ExposeAllKeys:     handleRequest.ExposeAllKeys,
		}, common.DecisionHandlers{
			GetCache: func(environmentID, id string) (*common.VisitorAssignments, error) {
				return handleRequest.DecisionContext.AssignmentsManager.LoadAssignments(environmentID, id)
			},
			SaveCache: func(environmentID, id string, assignment *common.VisitorAssignments) error {
				handleRequest.DecisionContext.AssignmentsManager.SaveAssignments(environmentID, id, assignment.Assignments, handleRequest.Time)
				return nil
			},
			ActivateCampaigns: func(activations []*common.VisitorActivation) error {
				// Initialize future campaign activations
				cActivations := []*models.CampaignActivation{}
				for _, a := range activations {
					cActivations = append(cActivations, &models.CampaignActivation{
						EnvID:       a.EnvironmentID,
						VisitorID:   a.AnonymousID,
						CustomerID:  a.VisitorID,
						CampaignID:  a.VariationGroupID,
						VariationID: a.VariationID,
						Timestamp:   handleRequest.Time.UnixNano() / 1000000,
					})
				}
				return handleRequest.DecisionContext.HitProcessor.TrackHits(connectors.TrackingHits{
					CampaignActivations: cActivations,
				})
			},
		})

	handleRequest.DecisionResponse = decisionResponse

	return err
}
