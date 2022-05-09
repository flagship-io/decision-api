package middlewares

import (
	"net/http"

	"github.com/flagship-io/decision-api/pkg/models"
)

func Version(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Flagship-Version", models.Version)
		handler(w, r)
	}
}
