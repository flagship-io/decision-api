package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecover(t *testing.T) {
	w := httptest.NewRecorder()
	Recover(true, func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})(w, &http.Request{})
	resp := w.Result()
	assert.Equal(t, 500, resp.StatusCode)
}
