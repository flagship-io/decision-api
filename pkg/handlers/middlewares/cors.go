package middlewares

import (
	"net/http"

	"github.com/flagship-io/decision-api/pkg/models"
)

func Cors(corsOptions *models.CorsOptions, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if corsOptions != nil && corsOptions.Enabled {
			w.Header().Set("Access-Control-Allow-Origin", corsOptions.AllowedOrigins)
			w.Header().Set("Access-Control-Allow-Headers", corsOptions.AllowedHeaders)
			if r.Method == "OPTIONS" {
				w.WriteHeader(200)
				return
			}
		}
		handler(w, r)
	}
}
