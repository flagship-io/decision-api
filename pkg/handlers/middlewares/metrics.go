package middlewares

import (
	"fmt"
	"net/http"
	"time"

	gokitexpvar "github.com/go-kit/kit/metrics/expvar"
)

type MetricsRegistry struct {
	responseTimes map[string]*gokitexpvar.Histogram
	errors        map[string]*gokitexpvar.Counter
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	// WriteHeader(int) is not called if our response implicitly returns 200 OK, so
	// we default to that status code.
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

var metrics *MetricsRegistry = &MetricsRegistry{
	responseTimes: make(map[string]*gokitexpvar.Histogram),
	errors:        make(map[string]*gokitexpvar.Counter),
}

func Metrics(name string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	if _, ok := metrics.responseTimes[name]; !ok {
		metrics.responseTimes[name] = gokitexpvar.NewHistogram(fmt.Sprintf("handlers.%s.response_time", name), 50)
	}
	if _, ok := metrics.errors[name]; !ok {
		metrics.errors[name] = gokitexpvar.NewCounter(fmt.Sprintf("handlers.%s.errors", name))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := NewLoggingResponseWriter(w)
		defer func(start time.Time) {
			metrics.responseTimes[name].Observe(float64(time.Since(start).Milliseconds()))
			if lrw.statusCode >= 500 {
				metrics.errors[name].Add(1)
			}
		}(start)
		handler(lrw, r)
	}
}
