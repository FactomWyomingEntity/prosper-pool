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

// OpenExchangeRatesDataSource is the datasource at "https://openexchangerates.org/"
type OpenExchangeRatesDataSource struct {
	apikey string
}

func NewOpenExchangeRatesDataSource(conf *viper.Viper) (*OpenExchangeRatesDataSource, error) {
	s := new(OpenExchangeRatesDataSource)
	s.apikey = conf.GetString(config.ConfigOpenExchangeRatesKey)
	if s.apikey == "" {
		return nil, fmt.Errorf("%s requires an api key", s.Name())
	}

	return s, nil
}

func (d *OpenExchangeRatesDataSource) Name() string {
	return "OpenExchangeRates"
}

func (d *OpenExchangeRatesDataSource) Url() string {
	return "https://openexchangerates.org/"
}

func (d *OpenExchangeRatesDataSource) SupportedPegs() []string {
	return MergeLists(CurrencyAssets, CommodityAssets, []string{"XBT"}, V4CurrencyAdditions)
}

func (d *OpenExchangeRatesDataSource) FetchPegPrices() (peg PegAssets, err error) {
	resp, err := d.CallOpenExchangeRates()
	if err != nil {
		return nil, err
	}

	peg = make(map[string]PegItem)

	timestamp := time.Unix(resp.Timestamp, 0)
	for _, currencyISO := range d.SupportedPegs() {
		// Price is inverted
		if v, ok := resp.Rates[currencyISO]; ok {
			peg[currencyISO] = PegItem{Value: 1 / v, When: timestamp, WhenUnix: timestamp.Unix()}
		}

		// Special case for btc
		if currencyISO == "XBT" {
			if v, ok := resp.Rates["BTC"]; ok {
				peg[currencyISO] = PegItem{Value: 1 / v, When: timestamp, WhenUnix: timestamp.Unix()}
			}
		}
	}

	return
}

func (d *OpenExchangeRatesDataSource) FetchPegPrice(peg string) (i PegItem, err error) {
	return FetchPegPrice(peg, d.FetchPegPrices)
}

// --

type OpenExchangeRatesResponse struct {
	Disclaimer  string             `json:"disclaimer"`
	License     string             `json:"license"`
	Timestamp   int64              `json:"timestamp"`
	Base        string             `json:"base"`
	Error       bool               `json:"error"`
	Status      int64              `json:"status"`
	Message     string             `json:"message"`
	Description string             `json:"description"`
	Rates       map[string]float64 `json:"rates"`
}

func (d *OpenExchangeRatesDataSource) CallOpenExchangeRates() (response OpenExchangeRatesResponse, err error) {
	var OpenExchangeRatesResponse OpenExchangeRatesResponse

	operation := func() error {
		resp, err := http.Get("https://openexchangerates.org/api/latest.json?app_id=" + d.apikey)
		if err != nil {
			log.WithError(err).Warning("Failed to get response from OpenExchangeRates")
			return err
		}

		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		} else if err = json.Unmarshal(body, &OpenExchangeRatesResponse); err != nil {
			return err
		}
		return nil
	}

	err = backoff.Retry(operation, PollingExponentialBackOff())
	// Price is inverted
	if err == nil {
		for k, v := range OpenExchangeRatesResponse.Rates {
			OpenExchangeRatesResponse.Rates[k] = v
		}
	}
	return OpenExchangeRatesResponse, err
}
