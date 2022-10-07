package validation

import (
	"github.com/flagship-io/flagship-proto/activate_request"
)

type ErrorResponse struct {
	Status string            `json:"status"`
	Errors map[string]string `json:"errors"`
}

func BuildErrorResponse(bodyError map[string]string) *ErrorResponse {
	return &ErrorResponse{
		Status: "error",
		Errors: bodyError,
	}
}

func CheckErrorBody(body *activate_request.ActivateRequest) *ErrorResponse {
	errorResponse := map[string]string{}
	if body.Cid == "" {
		errorResponse["cid"] = "Field is mandatory."
	}
	if body.Vid == "" {
		errorResponse["vid"] = "Field is mandatory."
	}
	if body.Vaid == "" {
		errorResponse["vaid"] = "Field is mandatory."
	}
	if body.Caid == "" {
		errorResponse["caid"] = "Field is mandatory."
	}
	if len(errorResponse) == 0 {
		return nil
	}
	return BuildErrorResponse(errorResponse)
}
