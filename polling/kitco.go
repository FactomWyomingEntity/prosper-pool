// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package polling

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

// KitcoDataSource is the datasource at "https://www.kitco.com/"
type KitcoDataSource struct {
}

func NewKitcoDataSource(_ *viper.Viper) (*KitcoDataSource, error) {
	s := new(KitcoDataSource)
	return s, nil
}

func (d *KitcoDataSource) Name() string {
	return "Kitco"
}

func (d *KitcoDataSource) Url() string {
	return "https://www.kitco.com/"
}

func (d *KitcoDataSource) SupportedPegs() []string {
	return CommodityAssets
}

func (d *KitcoDataSource) FetchPegPrices() (peg PegAssets, err error) {
	resp, err := CallKitcoWeb()
	if err != nil {
		return nil, err
	}

	peg = make(map[string]PegItem)

	peg["XAG"], err = d.parseData(resp.Silver)
	if err != nil {
		return nil, err
	}

	peg["XAU"], err = d.parseData(resp.Gold)
	if err != nil {
		return nil, err
	}

	peg["XPD"], err = d.parseData(resp.Palladium)
	if err != nil {
		return nil, err
	}

	peg["XPT"], err = d.parseData(resp.Platinum)
	if err != nil {
		return nil, err
	}

	return
}

func (d *KitcoDataSource) parseData(data KitcoRecord) (PegItem, error) {
	i := PegItem{}
	timestamp, err := time.Parse(d.dateFormat(), data.Date)
	if err != nil {
		return i, err
	}

	v, err := strconv.ParseFloat(data.Bid, 64)
	if err != nil {
		return i, err
	}

	return PegItem{Value: v, When: timestamp, WhenUnix: timestamp.Unix()}, nil
}

func (d *KitcoDataSource) dateFormat() string {
	return "01/02/2006"
}

func (d *KitcoDataSource) FetchPegPrice(peg string) (i PegItem, err error) {
	return FetchPegPrice(peg, d.FetchPegPrices)
}

// ---

type KitcoData struct {
	Silver    KitcoRecord
	Gold      KitcoRecord
	Platinum  KitcoRecord
	Palladium KitcoRecord
	Rhodium   KitcoRecord
}

type KitcoRecord struct {
	Date          string
	Tm            string
	Bid           string
	Ask           string
	Change        string
	PercentChange string
	Low           string
	High          string
}

func CallKitcoWeb() (KitcoData, error) {
	var kData KitcoData
	var emptyData KitcoData

	resp, err := http.Get("https://www.kitco.com/market/")
	if err != nil {
		log.WithError(err).Warning("Failed to get response from Kitco")
		return emptyData, err
	}

	defer resp.Body.Close()
	if body, err := ioutil.ReadAll(resp.Body); err != nil {
		return emptyData, err
	} else {
		matchStart := "<table class=\"world_spot_price\">"
		matchStop := "</table>"
		strResp := string(body)
		start := strings.Index(strResp, matchStart)
		if start < 0 {
			err = errors.New("No Response")
			log.WithError(err).Warning("Failed to get response from Kitco")
			return emptyData, err
		}
		strResp = strResp[start:]
		stop := strings.Index(strResp, matchStop)
		strResp = strResp[0 : stop+9]
		rows := strings.Split(strResp, "\n")
		for _, r := range rows {
			if strings.Index(r, "wsp-") > 0 {
				ParseKitco(r, &kData)
			}
		}
	}

	return kData, err
}

func ParseKitco(line string, kData *KitcoData) {

	if strings.Index(line, "wsp-AU-date") > 0 {
		kData.Gold.Date = PullValue(line, 1)
		//		fmt.Println("kData.Gold.Date:", kData.Gold.Date)
	} else if strings.Index(line, "wsp-AU-time") > 0 {
		kData.Gold.Tm = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AU-bid") > 0 {
		kData.Gold.Bid = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AU-ask") > 0 {
		kData.Gold.Ask = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AU-change") > 0 {
		kData.Gold.Change = PullValue(line, 2)
	} else if strings.Index(line, "wsp-AU-change-percent") > 0 {
		kData.Gold.PercentChange = PullValue(line, 2)
	} else if strings.Index(line, "wsp-AU-low") > 0 {
		kData.Gold.Low = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AU-high") > 0 {
		kData.Gold.High = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AG-date") > 0 {
		kData.Silver.Date = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AG-time") > 0 {
		kData.Silver.Tm = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AG-bid") > 0 {
		kData.Silver.Bid = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AG-ask") > 0 {
		kData.Silver.Ask = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AG-change") > 0 {
		kData.Silver.Change = PullValue(line, 2)
	} else if strings.Index(line, "wsp-AG-change-percent") > 0 {
		kData.Silver.PercentChange = PullValue(line, 2)
	} else if strings.Index(line, "wsp-AG-low") > 0 {
		kData.Silver.Low = PullValue(line, 1)
	} else if strings.Index(line, "wsp-AG-high") > 0 {
		kData.Silver.High = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PT-date") > 0 {
		kData.Platinum.Date = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PT-time") > 0 {
		kData.Platinum.Tm = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PT-bid") > 0 {
		kData.Platinum.Bid = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PT-ask") > 0 {
		kData.Platinum.Ask = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PT-change") > 0 {
		kData.Platinum.Change = PullValue(line, 2)
	} else if strings.Index(line, "wsp-PT-change-percent") > 0 {
		kData.Platinum.PercentChange = PullValue(line, 2)
	} else if strings.Index(line, "wsp-PT-low") > 0 {
		kData.Platinum.Low = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PT-high") > 0 {
		kData.Platinum.High = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PD-date") > 0 {
		kData.Palladium.Date = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PD-time") > 0 {
		kData.Palladium.Tm = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PD-bid") > 0 {
		kData.Palladium.Bid = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PD-ask") > 0 {
		kData.Palladium.Ask = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PD-change") > 0 {
		kData.Palladium.Change = PullValue(line, 2)
	} else if strings.Index(line, "wsp-PD-change-percent") > 0 {
		kData.Palladium.PercentChange = PullValue(line, 2)
	} else if strings.Index(line, "wsp-PD-low") > 0 {
		kData.Palladium.Low = PullValue(line, 1)
	} else if strings.Index(line, "wsp-PD-high") > 0 {
		kData.Palladium.High = PullValue(line, 1)
	} else if strings.Index(line, "wsp-RH-date") > 0 {
		kData.Rhodium.Date = PullValue(line, 1)
	} else if strings.Index(line, "wsp-RH-time") > 0 {
		kData.Rhodium.Tm = PullValue(line, 1)
	} else if strings.Index(line, "wsp-RH-bid") > 0 {
		kData.Rhodium.Bid = PullValue(line, 1)
	} else if strings.Index(line, "wsp-RH-ask") > 0 {
		kData.Rhodium.Ask = PullValue(line, 1)
	} else if strings.Index(line, "wsp-RH-change") > 0 {
		kData.Rhodium.Change = PullValue(line, 2)
	} else if strings.Index(line, "wsp-RH-change-percent") > 0 {
		kData.Rhodium.PercentChange = PullValue(line, 2)
	} else if strings.Index(line, "wsp-RH-low") > 0 {
		kData.Rhodium.Low = PullValue(line, 1)
	} else if strings.Index(line, "wsp-RH-high") > 0 {
		kData.Rhodium.High = PullValue(line, 1)
	}
}

func PullValue(line string, howMany int) string {
	i := 0
	//fmt.Println(line)
	var pos int
	for i < howMany {
		//find the end of the howmany-th tag
		pos = strings.Index(line, ">")
		line = line[pos+1:]
		//fmt.Println(line)
		i = i + 1
	}
	//fmt.Println("line:", line)
	pos = strings.Index(line, "<")
	//fmt.Println("POS:", pos)
	line = line[0:pos]
	//fmt.Println(line)
	return line
}
