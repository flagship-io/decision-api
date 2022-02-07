package utils

import (
	"encoding/json"
	"net/http"

	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/wrappers"
)

// ClientErrorMessage represents a bad request response
type ClientErrorMessage struct {
	Message string `json:"message"`
}

// WriteServerError returns a 500 Internal Server Error response
func WriteServerError(w http.ResponseWriter, err error) {
	body := &ClientErrorMessage{Message: err.Error()}
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

// WriteClientError similarly add a helper for send responses relating to client errors.
func WriteClientError(w http.ResponseWriter, status int, message string) {
	body := &ClientErrorMessage{Message: message}
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

// WriteJSONStringOk similarly add a helper to send json stringified responses with status OK.
func WriteJSONStringOk(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

// WriteJSONOk similarly add a helper to send json responses with status OK.
func WriteJSONOk(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)

	if err != nil {
		WriteServerError(w, err)
		return
	}
}

// WriteNoContent similarly add a helper to send 204 responses .
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(204)
}

// WritePanicResponse writes a response for panic mode
func WritePanicResponse(w http.ResponseWriter, visitorID *wrappers.StringValue) {
	ma := jsonpb.Marshaler{EmitDefaults: true}

	decisionResponse := decision_response.DecisionResponsePanic{}
	decisionResponse.Campaigns = []*decision_response.Campaign{}
	decisionResponse.VisitorId = visitorID
	decisionResponse.Panic = true

	err := ma.Marshal(w, &decisionResponse)

	if err != nil {
		WriteServerError(w, err)
	}
}
