package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	w := httptest.NewRecorder()
	Version(func(w http.ResponseWriter, r *http.Request) {
	})(w, &http.Request{})
	resp := w.Result()
	assert.Equal(t, models.Version, resp.Header.Get("x-flagship-version"))
}
