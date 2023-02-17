package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flagship-io/decision-api/pkg/config"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestCors(t *testing.T) {
	w := httptest.NewRecorder()
	Cors(&models.CorsOptions{
		Enabled:        false,
		AllowedOrigins: config.ServerCorsAllowedOrigins,
	}, func(w http.ResponseWriter, r *http.Request) {

	})(w, &http.Request{
		Method: "POST",
	})
	resp := w.Result()
	assert.Equal(t, "", resp.Header.Get("access-control-allow-origin"))
	assert.Equal(t, "", resp.Header.Get("access-control-allow-headers"))

	w = httptest.NewRecorder()
	overridenAllowedHeaders := "X-Overriden-Header"
	Cors(&models.CorsOptions{
		Enabled:        true,
		AllowedOrigins: "*",
		AllowedHeaders: overridenAllowedHeaders,
	}, func(w http.ResponseWriter, r *http.Request) {

	})(w, &http.Request{
		Method: "POST",
	})
	resp = w.Result()
	assert.Equal(t, "*", resp.Header.Get("access-control-allow-origin"))
	assert.Equal(t, overridenAllowedHeaders, resp.Header.Get("access-control-allow-headers"))

	w = httptest.NewRecorder()
	Cors(&models.CorsOptions{
		Enabled:        true,
		AllowedOrigins: "localhost",
	}, func(w http.ResponseWriter, r *http.Request) {

	})(w, &http.Request{
		Method: "OPTIONS",
	})
	resp = w.Result()
	assert.Equal(t, "localhost", resp.Header.Get("access-control-allow-origin"))
	assert.Equal(t, 200, resp.StatusCode)
}
