package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestCors(t *testing.T) {
	w := httptest.NewRecorder()
	Cors(&models.CorsOptions{
		Enabled:        true,
		AllowedOrigins: "*",
	}, func(w http.ResponseWriter, r *http.Request) {

	})(w, &http.Request{
		Method: "POST",
	})
	resp := w.Result()
	assert.Equal(t, "*", resp.Header.Get("access-control-allow-origin"))

	w = httptest.NewRecorder()
	Cors(&models.CorsOptions{
		Enabled:        true,
		AllowedOrigins: "*",
	}, func(w http.ResponseWriter, r *http.Request) {

	})(w, &http.Request{
		Method: "OPTIONS",
	})
	resp = w.Result()
	assert.Equal(t, "*", resp.Header.Get("access-control-allow-origin"))
	assert.Equal(t, 200, resp.StatusCode)
}
