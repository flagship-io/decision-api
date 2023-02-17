package middlewares

import (
	"fmt"
	"net/http"

	"github.com/flagship-io/decision-api/internal/reswriter"
)

func Recover(enabled bool, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if enabled {
			defer func() {
				if err := recover(); err != nil {
					reswriter.WriteServerError(w, fmt.Errorf("unexpected error occurred: %v", err))
				}
			}()
		}

		handler(w, r)
	}
}
