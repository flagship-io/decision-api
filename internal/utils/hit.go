package utils

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"gitlab.com/abtasty/protobuf/ptypes.git/hit"
	"gitlab.com/abtasty/protobuf/ptypes.git/hit_request"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var defaultUrl = "https://ariane.abtasty.com"
var marshaller = jsonpb.Marshaler{}

// BuildSegmentHit builds a segment hit from envID, visitorID and segment map
func BuildSegmentHit(envID string, visitorID string, customerID string, segments map[string]interface{}, timestamp int64) *hit_request.HitRequest {
	segmentsString := map[string]string{}
	for k, v := range segments {
		segmentsString[k] = fmt.Sprintf("%v", v)
	}

	hit := &hit_request.HitRequest{
		Cid: wrapperspb.String(envID),
		Vid: wrapperspb.String(visitorID),
		S:   segmentsString,
		T:   hit.Hit_SEGMENT,
	}

	if customerID != "" {
		hit.Cuid = wrapperspb.String(customerID)
	}

	if timestamp > 0 {
		// Get milliseconds of queuetime
		hit.Qt = wrapperspb.Int64((time.Now().UnixNano()/1000000 - timestamp))
	}

	return hit
}

// BuildCampaignHit creates a campaign hit
func BuildCampaignHit(clientID string, visitorID string, customerID string, variationGroupID string, variationID string, timestamp int64) *hit_request.HitRequest {
	hit := &hit_request.HitRequest{
		Cid:  wrapperspb.String(clientID),
		Vid:  wrapperspb.String(visitorID),
		Caid: wrapperspb.String(variationGroupID),
		Vaid: wrapperspb.String(variationID),
		T:    hit.Hit_CAMPAIGN,
	}

	if customerID != "" {
		hit.Cuid = wrapperspb.String(customerID)
	}

	if timestamp > 0 {
		// Get milliseconds of queuetime
		hit.Qt = wrapperspb.Int64((time.Now().UnixNano()/1000000 - timestamp))
	}

	return hit
}

func SendBatchHit(innerHits []*hit_request.HitRequest) error {
	batchHit := &hit_request.HitRequest{
		T:  hit.Hit_BATCH,
		Ds: wrapperspb.String("APP"),
		H:  innerHits,
	}
	return SendHit(batchHit)
}

// SendHit sends a batch hit to data collect
func SendHit(hit *hit_request.HitRequest) error {
	var url = os.Getenv("CB_DECISION_COLLECT_URL")
	if url == "" {
		url = defaultUrl
	}

	var (
		retries int = 3
		client      = &http.Client{
			Timeout: time.Second * 1,
		}
		jsonBody string
		err      error
	)

	for retries > 0 {
		jsonBody, err = marshaller.MarshalToString(hit)
		if err != nil {
			return err
		}

		_, err = client.Post(url, "application/json", bytes.NewBuffer([]byte(jsonBody)))

		if err == nil {
			break
		}
		log.Printf("HTTP error occured. Retrying %v time. %v", retries-1, err.Error())
		retries--
	}

	if err == nil {
		log.Printf("[ARIANE] Sent hit on %s: %s", url, string(jsonBody))
	} else {
		log.Printf("[ARIANE] [ERR] Error when sending hit: %v", err)
	}

	return err
}
