package apilogic

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildHandleRequestHasCorrectSendContextEvent(t *testing.T) {
	tests := map[string]struct {
		queryVisitorConsent string
		bodyVisitorConsent  string
		result              bool
	}{
		"VisitorConsentEmpty":         {"", "", true},
		"BodyVisitorConsentFalse":     {"\"visitor_consent\": false,", "", false},
		"BodyVisitorConsentTrue":      {"\"visitor_consent\": true,", "", true},
		"QueryVisitorConsentFalse":    {"", "sendContextEvent=false", false},
		"QueryVisitorConsentTrue":     {"", "sendContextEvent=true", true},
		"VisitorConsentBothTrue":      {"\"visitor_consent\": true,", "sendContextEvent=true", true},
		"VisitorConsentBothFalse":     {"\"visitor_consent\": false,", "sendContextEvent=false", false},
		"VisitorConsentBothDifferent": {"\"visitor_consent\": true,", "sendContextEvent=false", false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			body := `{
				"visitor_id": "123",
				` + test.queryVisitorConsent + `
				"context": {}
			}`

			req := &http.Request{
				URL: &url.URL{
					RawQuery: test.bodyVisitorConsent,
				},
				Body:   io.NopCloser(strings.NewReader(body)),
				Method: "POST",
			}

			hr, err := BuildHandleRequest(req)

			assert.NotNil(t, hr)
			assert.Nil(t, err)
			assert.Equal(t, test.result, hr.SendContextEvent)
		})
	}

}

func TestBuildHandleRequestHasCorrectExposeAllKeys(t *testing.T) {
	tests := map[string]struct {
		path   string
		query  string
		result bool
	}{
		"FlagRouteUrlParamEmpty":      {"/v2/flags", "", true},
		"FlagRouteUrlParamTrue":       {"/v2/flags", "exposeAllKeys=true", true},
		"FlagRouteUrlParamFalse":      {"/v2/flags", "exposeAllKeys=false", false},
		"CampaignsRouteUrlParamFalse": {"/v2/campaigns", "", false},
		"CampaignsRouteUrlParamTrue":  {"/v2/campaigns", "exposeAllKeys=true", true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			body := `{
				"visitor_id": "123",
				"context": {}
			}`

			req := &http.Request{
				URL: &url.URL{
					Path:     test.path,
					RawQuery: test.query,
				},
				Body:   io.NopCloser(strings.NewReader(body)),
				Method: "POST",
			}

			hr, err := BuildHandleRequest(req)

			assert.NotNil(t, hr)
			assert.Nil(t, err)
			assert.Equal(t, test.result, hr.ExposeAllKeys)
		})
	}

}
