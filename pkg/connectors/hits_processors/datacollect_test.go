package hits_processors

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/flagship-io/decision-api/pkg/connectors"
	"github.com/flagship-io/decision-api/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDataCollectBuilder(t *testing.T) {
	httpClient := &http.Client{}
	dc := NewDataCollectProcessor(
		WithBatchOptions(50, time.Second),
		WithLogger("debug", "json"),
		WithTrackingURL("https://tracking-url.dev"),
		WithHTTPClient(httpClient))

	assert.Equal(t, 50, dc.batchSize)
	assert.Equal(t, time.Second, dc.batchingWindow)
	assert.Equal(t, logrus.DebugLevel, dc.logger.Logger.Level)
	assert.Equal(t, "https://tracking-url.dev", dc.trackingURL)
	assert.Equal(t, httpClient, dc.httpClient)
}

func TestDataCollectTrack(t *testing.T) {
	lock := &sync.Mutex{}
	var bodySents []string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		lock.Lock()
		lastBodySent, _ := io.ReadAll(req.Body)
		bodySents = append(bodySents, string(lastBodySent))
		_, err := rw.Write([]byte("{}"))
		assert.Nil(t, err)
		lock.Unlock()
	}))
	// Close the server when test finishes
	defer server.Close()

	dcProcessor := NewDataCollectProcessor(WithBatchOptions(2, 100*time.Millisecond), WithTrackingURL(server.URL))
	ts := time.Now().Add(-1 * time.Second).UnixMilli()

	err := dcProcessor.TrackHits(connectors.TrackingHits{
		CampaignActivations: []*models.CampaignActivation{{
			EnvID:       "env_id",
			CustomerID:  "customer_id",
			VisitorID:   "visitor_id",
			CampaignID:  "campaign_id",
			VariationID: "variation_id",
			Timestamp:   ts,
		}},
		VisitorContext: []*models.VisitorContext{{
			EnvID:      "env_id",
			VisitorID:  "visitor_id",
			CustomerID: "customer_id",
			Partner:    "partner",
			Context:    map[string]interface{}{"key": "value"},
			Timestamp:  ts,
		}},
	})

	time.Sleep(110 * time.Millisecond)
	assert.Nil(t, err)
	lock.Lock()
	assert.Equal(t, 1, len(bodySents))

	batch := &batchHit{}
	err = json.Unmarshal([]byte(bodySents[0]), batch)
	lock.Unlock()
	assert.Nil(t, err)
	assert.Equal(t, "BATCH", batch.Type)
	assert.Equal(t, "APP", batch.DataSource)
	assert.Equal(t, 2, len(batch.Hits))
	assert.Equal(t, "env_id", batch.Hits[0]["cid"])
	assert.Equal(t, "customer_id", batch.Hits[0]["cuid"])
	assert.Equal(t, "visitor_id", batch.Hits[0]["vid"])
	assert.Equal(t, "CAMPAIGN", batch.Hits[0]["t"])
	assert.Equal(t, "campaign_id", batch.Hits[0]["caid"])
	assert.Equal(t, "variation_id", batch.Hits[0]["vaid"])
	assert.True(t, batch.Hits[0]["qt"].(float64) < 1010 && batch.Hits[0]["qt"].(float64) >= 1000)

	assert.Equal(t, "env_id", batch.Hits[1]["cid"])
	assert.Equal(t, "visitor_id", batch.Hits[1]["vid"])
	assert.Equal(t, "customer_id", batch.Hits[1]["cuid"])
	assert.Equal(t, "SEGMENT", batch.Hits[1]["t"])
	assert.EqualValues(t, map[string]interface{}{
		"key": "value",
	}, batch.Hits[1]["s"])
	assert.True(t, batch.Hits[1]["qt"].(float64) < 1010 && batch.Hits[1]["qt"].(float64) >= 1000)
}
