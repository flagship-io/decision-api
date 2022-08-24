package apilogic

import (
	"net/http"
	"strings"

	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/internal/utils"
	"github.com/flagship-io/flagship-common/targeting"
)

// BuildHandleRequest builds a handle.Request object from the API Gateway request
func BuildHandleRequest(req *http.Request) (*handle.Request, error) {
	handleRequest := handle.NewRequestFromHTTP(req)
	decisionRequest, err := utils.GetDecisionRequest(req)

	if err != nil {
		return nil, err
	}

	handleRequest.Mode = "normal"
	mode := req.URL.Query().Get("mode")
	if mode != "" {
		handleRequest.Mode = mode
	}

	// exposeAllKeys default true for flags route
	if strings.Contains(req.URL.String(), "/flags") {
		handleRequest.ExposeAllKeys = true
	}
	// exposeAllKeys url param extend
	exposeAllKeys := req.URL.Query().Get("exposeAllKeys")
	if exposeAllKeys != "" {
		handleRequest.ExposeAllKeys = exposeAllKeys == "true"
	}

	hasVisitorConsented := decisionRequest.VisitorConsent == nil || decisionRequest.VisitorConsent.GetValue()
	handleRequest.SendContextEvent = req.URL.Query().Get("sendContextEvent") != "false" && hasVisitorConsented
	handleRequest.DecisionRequest = decisionRequest
	handleRequest.FullVisitorContext = &targeting.Context{
		Standard:             decisionRequest.GetContext(),
		IntegrationProviders: map[string]targeting.ContextMap{},
	}
	handleRequest.Extras = req.URL.Query()["extras"]

	return &handleRequest, nil
}
