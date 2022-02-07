package apilogic

import (
	"net/http"

	"github.com/flagship-io/decision-api/internal/handle"
	"github.com/flagship-io/decision-api/internal/utils"
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

	exposeAllKeys := req.URL.Query().Get("exposeAllKeys")
	if exposeAllKeys != "" {
		handleRequest.ExposeAllKeys = exposeAllKeys == "true"
	}

	sendContextEvent := req.URL.Query().Get("sendContextEvent")
	if sendContextEvent != "" {
		handleRequest.SendContextEvent = exposeAllKeys != "false"
	}

	handleRequest.DecisionRequest = decisionRequest
	handleRequest.Extras = req.URL.Query()["extras"]

	return &handleRequest, nil
}
