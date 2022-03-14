package middlewares

import (
	"expvar"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	w := httptest.NewRecorder()
	Metrics("test", func(w http.ResponseWriter, r *http.Request) {})(w, &http.Request{})
	w.Result()
	assert.NotNil(t, metrics.responseTimes["test"])
	v := reflect.ValueOf(metrics.responseTimes["test"])
	p99 := reflect.Indirect(v).FieldByName("p99")
	assert.False(t, p99.IsNil())

	w = httptest.NewRecorder()
	Metrics("test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})(w, &http.Request{})
	w.Result()
	assert.NotNil(t, metrics.errors["test"])
	v = reflect.ValueOf(metrics.errors["test"])
	f := reflect.Indirect(v).FieldByName("f")
	expvarFloat := f.Convert(reflect.TypeOf(&expvar.Float{}))
	assert.Equal(t, 1, expvarFloat.Float())
}
