package utils

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// WriteServerError returns a 500 Internal Server Error response
func TestWriteServerError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteServerError(w, errors.New("test_error"))
	resp := w.Result()
	json, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "{\"message\":\"test_error\"}\n", string(json))
}

// WriteClientError similarly add a helper for send responses relating to client errors.
func TestWriteClientError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteClientError(w, 400, "test_error")
	resp := w.Result()
	json, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, "{\"message\":\"test_error\"}\n", string(json))
}

// WriteJSONStringOk similarly add a helper to send json stringified responses with status OK.
func TestWriteJSONStringOk(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONStringOk(w, `{"status":"ok"}`)
	resp := w.Result()
	json, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"status":"ok"}`, string(json))
}

func TestWriteJSONOk(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSONOk(w, map[string]interface{}{"key": "value"})
	resp := w.Result()
	json, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "{\"key\":\"value\"}\n", string(json))
}

func TestWriteNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	WriteNoContent(w)
	resp := w.Result()
	json, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, 204, w.Code)
	assert.Equal(t, "", string(json))
}

func TestWritePanicResponse(t *testing.T) {
	w := httptest.NewRecorder()
	WritePanicResponse(w, wrapperspb.String("visitor_id"))
	resp := w.Result()
	json, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "{\"visitorId\":\"visitor_id\",\"campaigns\":[],\"panic\":true}", strings.ReplaceAll(string(json), " ", ""))
}
