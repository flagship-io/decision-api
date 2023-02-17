package udc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type UDCVisitorRow struct {
	Segment string `json:"segment"`
	Value   string `json:"value"`
	Partner string `json:"partner"`
}

const UDC_TIMEOUT = 1000

var udcUrl string = "https://api-data-connector.flagship.io"

func FetchVisitorData(environmentID string, visitorID string) ([]UDCVisitorRow, error) {
	if udcUrl == "" {
		return nil, errors.New("missing UDC_URL env variable")
	}

	url := fmt.Sprintf("%s/accounts/%s/segments/%s", udcUrl, environmentID, visitorID)

	httpClient := &http.Client{
		Timeout: time.Duration(UDC_TIMEOUT) * time.Millisecond,
	}
	r, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var data []UDCVisitorRow
	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		return nil, errors.New("fetchVisitorData json decode error : " + err.Error())
	}

	return data, nil
}

func SetURL(url string) {
	udcUrl = url
}
