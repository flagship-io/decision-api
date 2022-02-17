package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flagship-io/flagship-proto/decision_request"
	"google.golang.org/protobuf/encoding/protojson"
)

const apiKeyURLParam = "token"

// GetAPIKeyURLParam returns the name of the url param when passed by url
func GetAPIKeyURLParam() string {
	return apiKeyURLParam
}

// GetDecisionRequest transforms http request into a DecisionRequest
func GetDecisionRequest(r *http.Request) (*decision_request.DecisionRequest, error) {
	decisionRequest, err := unmarshalHit(r)

	if err != nil {
		//raven.CaptureError(err, nil)
		return nil, err
	}

	return decisionRequest, nil
}

func unmarshalHit(r *http.Request) (*decision_request.DecisionRequest, error) {
	if r.Method == http.MethodPost {
		return unmarshalPost(r)
	} else if r.Method == http.MethodGet {
		return unmarshalGet(r)
	}
	return nil, errors.New("the hit is not formatted correctly")
}

func parseJSONBody(data []byte) (*decision_request.DecisionRequest, error) {
	decisionRequest := &decision_request.DecisionRequest{}
	if err := protojson.Unmarshal(data, decisionRequest); err != nil {
		switch err.(type) {
		case *json.UnmarshalTypeError:
			return nil, fmt.Errorf("syntax error in body json request. Field type incorrect : %s", err.Error())
		default:
			if strings.Contains(err.Error(), "unknown field") {
				return nil, fmt.Errorf("json body is not valid. %s", err.Error())
			}
			return nil, fmt.Errorf("syntax error in body json request. Must be a valid json : %s", err.Error())
		}
	}
	return decisionRequest, nil
}

func unmarshalGet(r *http.Request) (*decision_request.DecisionRequest, error) {
	if len(r.URL.Query()) == 0 {
		return nil, errors.New("empty http query")
	}

	// Do not parse token query string
	cleanedQv := map[string]string{}
	for k := range r.URL.Query() {
		if k != GetAPIKeyURLParam() {
			cleanedQv[k] = r.URL.Query().Get(k)
		}
	}

	j, _ := json.Marshal(formatQuery(cleanedQv))

	return parseJSONBody(j)
}

func unmarshalPost(r *http.Request) (*decision_request.DecisionRequest, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return parseJSONBody(data)
}
