package middlewares

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/utils/logger"
	"github.com/stretchr/testify/assert"
)

type TestHandler struct{}

func (*TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(100 * time.Millisecond)
	w.WriteHeader(200)
}

func TestRequestLogger(t *testing.T) {
	w := httptest.NewRecorder()
	var b bytes.Buffer
	mockWriter := bufio.NewWriter(&b)
	logger := logger.New("debug", logger.FORMAT_TEXT, "test")
	logger.Logger.SetOutput(mockWriter)

	url, _ := url.Parse("http://fakeurl.com")
	r := &http.Request{
		URL:        url,
		Method:     "GET",
		RemoteAddr: "127.0.0.1",
	}
	RequestLogger(logger, &TestHandler{}).ServeHTTP(w, r)

	mockWriter.Flush()
	ret := b.Bytes()
	assert.Contains(t, string(ret), r.RemoteAddr)
	assert.Contains(t, string(ret), "200")
	assert.Contains(t, string(ret), "100")
	assert.Contains(t, string(ret), url.Host)
}
