package reswriter

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/flagship-io/flagship-proto/decision_response"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	jsonErr := json.NewEncoder(w).Encode(body)
	if jsonErr != nil {
		log.Printf("error when encoding body: %v", jsonErr)
	}
}

// WriteClientError similarly add a helper for send responses relating to client errors.
func WriteClientError(w http.ResponseWriter, status int, message string) {
	body := &ClientErrorMessage{Message: message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	jsonErr := json.NewEncoder(w).Encode(body)
	if jsonErr != nil {
		log.Printf("error when encoding body: %v", jsonErr)
	}
}

// WriteJSONStringOk similarly add a helper to send json stringified responses with status OK.
func WriteJSONStringOk(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	_, writeErr := w.Write([]byte(data))
	if writeErr != nil {
		log.Printf("error when writing body: %v", writeErr)
	}
}

// WriteJSONOk similarly add a helper to send json responses with status OK.
func WriteJSONOk(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
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
func WritePanicResponse(w http.ResponseWriter, visitorID *wrapperspb.StringValue) {
	decisionResponse := decision_response.DecisionResponsePanic{}
	decisionResponse.Campaigns = []*decision_response.Campaign{}
	decisionResponse.VisitorId = visitorID
	decisionResponse.Panic = true

	marshalOptions := protojson.MarshalOptions{
		EmitUnpopulated: true,
	}
	data, err := marshalOptions.Marshal(&decisionResponse)
	if err != nil {
		WriteServerError(w, err)
		return
	}
	_, writeErr := w.Write(data)
	if writeErr != nil {
		log.Printf("error when writing body: %v", writeErr)
	}
}
