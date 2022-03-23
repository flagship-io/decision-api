package middlewares

import (
	"net/http"
	"time"

	"github.com/flagship-io/decision-api/pkg/utils/logger"
)

func RequestLogger(logger *logger.Logger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := NewLoggingResponseWriter(w)
		defer func(start time.Time) {
			logger.Infof("%s | %d | %s | %v | %s", r.RemoteAddr, lrw.statusCode, r.Method, time.Since(start), r.URL)
		}(start)
		handler.ServeHTTP(lrw, r)
	})
}
