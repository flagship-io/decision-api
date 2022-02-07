package handlers

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/flagship-io/decision-api/internal/models"
	"github.com/flagship-io/flagship-proto/decision_response"
	"github.com/golang/protobuf/jsonpb"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	godotenv.Load("../../env.test")

	code := m.Run()
	os.Exit(code)
}

func TestCampaignsAssignment(t *testing.T) {
	url, _ := url.Parse("/campaigns?mode=full&sendContextEvent=false")
	body := `{"visitor_id": "1234", "context": {}, "trigger_hit": false }`
	w := httptest.NewRecorder()

	req := &http.Request{
		URL:    url,
		Body:   io.NopCloser(strings.NewReader(body)),
		Method: "POST",
	}

	Campaigns(&models.DecisionContext{
		EnvID:  "env_id_1",
		APIKey: "api_key_id",
	})(w, req)

	resp := w.Result()

	data := decision_response.DecisionResponseFull{}
	err := jsonpb.Unmarshal(resp.Body, &data)

	if err != nil {
		log.Fatalln("error unmarshaling campaigns.", err)
	}

	if data.VisitorId.Value != "1234" {
		t.Errorf("Wrong visitor ID : %v instead of 1234", data.VisitorId.Value)
	}

	// test campaign variations
	for _, c := range data.Campaigns {
		if c.Id.Value == "campaign_1" {
			assert.Equal(t, "vg_2", c.VariationGroupId.Value)
		} else if c.Id.Value == "image" {
			assert.Equal(t, "vg_1", c.VariationGroupId.Value)
		}
	}

	// test merged modifications
	assert.Equal(t, data.MergedModifications.Fields["testString"].AsInterface(), "string")
	assert.Equal(t, data.MergedModifications.Fields["testBool"].AsInterface(), true)
	assert.Equal(t, data.MergedModifications.Fields["testNumber"].AsInterface(), 11.)
	assert.Equal(t, data.MergedModifications.Fields["testWhatever"].AsInterface(), []interface{}{"a", 1.})
}
