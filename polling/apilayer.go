// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package polling

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/FactomWyomingEntity/prosper-pool/config"
	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// APILayerDataSource is the datasource at http://www.apilayer.net
type APILayerDataSource struct {
	apikey string
}

func NewAPILayerDataSource(conf *viper.Viper) (*APILayerDataSource, error) {
	s := new(APILayerDataSource)
	s.apikey = conf.GetString(config.ConfigApiLayerKey)
	if s.apikey == "" {
		return nil, fmt.Errorf("%s requires an api key", s.Name())
	}
	return s, nil
}

func (d *APILayerDataSource) Name() string {
	return "APILayer"
}

func (d *APILayerDataSource) Url() string {
	return "https://apilayer.com/"
}

func (d *APILayerDataSource) SupportedPegs() []string {
	return MergeLists(CurrencyAssets, V4CurrencyAdditions)
}

func (d *APILayerDataSource) FetchPegPrices() (peg PegAssets, err error) {
	resp, err := d.CallAPILayer()
	if err != nil {
		return nil, err
	}

	peg = make(map[string]PegItem)

	timestamp := time.Unix(resp.Timestamp, 0)
	for _, currencyISO := range d.SupportedPegs() {
		// Search for USDXXX pairs
		if v, ok := resp.Quotes["USD"+currencyISO]; ok {
			peg[currencyISO] = PegItem{Value: 1 / v, When: timestamp, WhenUnix: timestamp.Unix()}
		}
	}

	return
}

func (d *APILayerDataSource) FetchPegPrice(peg string) (i PegItem, err error) {
	return FetchPegPrice(peg, d.FetchPegPrices)
}

// ----

type APILayerResponse struct {
	Success   bool               `json:"success"`
	Terms     string             `json:"terms"`
	Privacy   string             `json:"privacy"`
	Timestamp int64              `json:"timestamp"`
	Source    string             `json:"source"`
	Quotes    map[string]float64 `json:"quotes"`
	Error     APILayerError      `json:"error"`
}

type APILayerError struct {
	Code int64
	Type string
	Info string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func (d *APILayerDataSource) CallAPILayer() (response APILayerResponse, err error) {
	var APILayerResponse APILayerResponse

	operation := func() error {
		resp, err := http.Get("http://www.apilayer.net/api/live?access_key=" + d.apikey)
		if err != nil {
			log.WithError(err).Warning("Failed to get response from API Layer")
		}

		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		} else if err = json.Unmarshal(body, &APILayerResponse); err != nil {
			return err
		}
		return err
	}

	err = backoff.Retry(operation, PollingExponentialBackOff())
	return APILayerResponse, err
}
