package middlewares

import (
	"expvar"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	w := httptest.NewRecorder()
	Metrics("test", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
	})(w, &http.Request{})
	w.Result()
	assert.NotNil(t, metrics.responseTimes["test"])
	assert.GreaterOrEqual(t, "10", expvar.Get("handlers.test.response_time.p50").String())
	assert.GreaterOrEqual(t, "10", expvar.Get("handlers.test.response_time.p90").String())
	assert.GreaterOrEqual(t, "10", expvar.Get("handlers.test.response_time.p95").String())
	assert.GreaterOrEqual(t, "10", expvar.Get("handlers.test.response_time.p99").String())

	w = httptest.NewRecorder()
	Metrics("test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})(w, &http.Request{})
	w.Result()
	assert.NotNil(t, metrics.errors["test"])
	assert.Equal(t, "1", expvar.Get("handlers.test.errors").String())
}
