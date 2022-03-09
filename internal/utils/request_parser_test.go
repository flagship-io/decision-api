package utils

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDecisionRequest(t *testing.T) {
	_, err := GetDecisionRequest(&http.Request{
		Method: "GET",
	})
	assert.Equal(t, errors.New("only POST http method is allowed"), err)

	_, err = GetDecisionRequest(&http.Request{
		Method: "POST",
		Body:   io.NopCloser(strings.NewReader("")),
	})
	assert.Contains(t, err.Error(), "Must be a valid json")

	_, err = GetDecisionRequest(&http.Request{
		Method: "POST",
		Body:   io.NopCloser(strings.NewReader("{\"wrong_key\":true}")),
	})
	assert.Contains(t, err.Error(), "json body is not valid")

	r, err := GetDecisionRequest(&http.Request{
		Method: "POST",
		Body:   io.NopCloser(strings.NewReader("{}")),
	})
	assert.Nil(t, err)
	assert.Equal(t, "", r.VisitorId.GetValue())
}
