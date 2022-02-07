package validation

import "gitlab.com/canarybay/protobuf/ptypes.git/event_request"

type EventErrorResponse struct {
	Status string            `json:"status"`
	Errors map[string]string `json:"errors"`
}

func BuildEventErrorResponse(bodyError map[string]string) *ErrorResponse {
	return &ErrorResponse{
		Status: "error",
		Errors: bodyError,
	}
}

func CheckEventErrorBody(body *event_request.EventRequest) *ErrorResponse {
	errorResponse := map[string]string{}
	if body.VisitorId.Value == "" {
		errorResponse["visitorId"] = "Field is mandatory."
	}
	if body.Type == event_request.EventRequest_NULL {
		errorResponse["type"] = "Field is mandatory."
	}
	if len(errorResponse) == 0 {
		return nil
	}
	return BuildEventErrorResponse(errorResponse)
}
