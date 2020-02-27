package polling_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/FactomWyomingEntity/prosper-pool/config"
	. "github.com/FactomWyomingEntity/prosper-pool/polling"
	"github.com/spf13/viper"
)

func TestCorrectCasing(t *testing.T) {
	if CorrectCasing("pegnetmarketcap") != "PegnetMarketCap" {
		t.Errorf("failed correct casing")
	}
}

// TestBasicPollingSources creates 8 polling sources. 5 have 1 asset, 3 have all.
// We then check to make sure the 5 sources are higher priority then the second 3.
// The second two have a priority order as well, and will be listed in the prioriy list
// for every asset.
func TestBasicPollingSources(t *testing.T) {
	end := 6
	// Create the unit test creator
	NewTestingDataSource = func(conf *viper.Viper, source string) (IDataSource, error) {
		s := new(UnitTestDataSource)
		v, err := strconv.Atoi(string(source[8]))
		if err != nil {
			panic(err)
		}
		s.Value = float64(v)
		s.Assets = []string{AllAssets[v]}
		s.SourceName = fmt.Sprintf("UnitTest%d", v)

		// Catch all
		if v >= end {
			s.Assets = AllAssets[1:]
		}
		return s, nil
	}

	conf := GetConfig(`
[oracledatasources]
  UnitTest1=1
  UnitTest2=2
  UnitTest3=3
  UnitTest4=4
  UnitTest5=5
  UnitTest6=6
  UnitTest7=7
  UnitTest8=8
`)

	s := NewDataSources(conf, false)

	pa, err := s.PullAllPEGAssets(1)
	if err != nil {
		t.Error(err)
	}
	for i, asset := range AllAssets {
		v, ok := pa[asset]
		if !ok {
			t.Errorf("%s is missing", asset)
			continue
		}
		if i < end {
			if int(v.Value) != i {
				t.Errorf("Exp value %d, found %d for %s", i, int(v.Value), asset)
			}

			// Let's also check there is 4 sources
			if len(s.AssetSources[asset]) != 4 && asset != "PEG" {
				t.Errorf("exp %d sources for %s, found %d", 4, asset, len(s.AssetSources[asset]))
			}
		} else {
			if int(v.Value) != end {
				t.Errorf("Exp value %d, found %d for %s", end, int(v.Value), asset)
			}
			// Let's also check there is 3 sources
			if len(s.AssetSources[asset]) != 3 {
				t.Errorf("exp %d sources for %s, found %d", 3, asset, len(s.AssetSources[asset]))
			}
		}
	}

	// Test the override mechanic
	t.Run("Test the override", func(t *testing.T) {
		conf := GetConfig(`
[oracledatasources]
  UnitTest1 = 1
  UnitTest2 = 2
  UnitTest3 = 3
  UnitTest4 = 4
  UnitTest5 = 5
  UnitTest6 = 6
  UnitTest7 = 7
  UnitTest8 = 8
[oracleassetdatasourcespriority]
  usd = "UnitTest8"

`)

		s = NewDataSources(conf, false)
		pa, err := s.PullAllPEGAssets(1)
		if err != nil {
			t.Error(err)
		}

		if v, ok := pa["USD"]; !ok {
			t.Error("No usd")
		} else {
			if int(v.Value) != 8 {
				t.Error("Override failed")
			}
		}

		if len(s.AssetSources["USD"]) != 1 {
			t.Errorf("exp %d sources for %s, found %d", 1, "USD", len(s.AssetSources["USD"]))
		}

		if s.AssetSources["USD"][0] != "UnitTest8" {
			t.Errorf("Exp UnitTest8, got %s", s.AssetSources["USD"][0])
		}
	})
}

func TestTruncate(t *testing.T) {
	type Vector struct {
		Vector float64
		Exp4   float64
		Exp8   float64
	}
	vects := []Vector{
		{1, 1, 1},
		{1.123456789, 1.1234, 1.12345678},
		{1.12, 1.12, 1.12},
		{1.1267, 1.1267, 1.1267},
	}

	for _, v := range vects {
		if r := TruncateTo4(v.Vector); r != v.Exp4 {
			t.Errorf("t4 exp %f, got %f", v.Exp4, r)
		}
		if r := TruncateTo8(v.Vector); r != v.Exp8 {
			t.Errorf("t8 exp %f, got %f", v.Exp8, r)
		}
	}
}

// AlwaysOnePolling returns 1 for all prices
func AlwaysOnePolling() *DataSources {
	NewTestingDataSource = func(conf *viper.Viper, source string) (IDataSource, error) {
		s := new(UnitTestDataSource)
		v, err := strconv.Atoi(string(source[8]))
		if err != nil {
			panic(err)
		}
		s.Value = float64(1)
		s.Assets = AllAssets
		s.SourceName = fmt.Sprintf("UnitTest%d", v)

		return s, nil
	}

	c := GetConfig(`
[oracledatasources]
  UnitTest1 = 1
`)

	s := NewDataSources(c, false)
	return s
}

// UnitTestDataSource just reports the Value for the supported Assets
type UnitTestDataSource struct {
	Value      float64
	Assets     []string
	SourceName string
}

func NewUnitTestDataSource(_ *viper.Viper) (*UnitTestDataSource, error) {
	s := new(UnitTestDataSource)
	return s, nil
}

func (d *UnitTestDataSource) Name() string {
	return d.SourceName
}

func (d *UnitTestDataSource) Url() string {
	return "https://unit.test/"
}

func (d *UnitTestDataSource) SupportedPegs() []string {
	return d.Assets
}

func (d *UnitTestDataSource) FetchPegPrices() (peg PegAssets, err error) {
	peg = make(map[string]PegItem)

	timestamp := time.Now()
	for _, asset := range d.SupportedPegs() {
		peg[asset] = PegItem{Value: d.Value, When: timestamp, WhenUnix: timestamp.Unix()}
	}

	return peg, nil
}

func (d *UnitTestDataSource) FetchPegPrice(peg string) (i PegItem, err error) {
	return FetchPegPrice(peg, d.FetchPegPrices)
}

func GetConfig(content string) *viper.Viper {
	var buf bytes.Buffer
	// The order of these retrieved is random since the settings are a map
	buf.WriteString(content)
	conf := viper.New()
	config.SetDefaults(conf)
	conf.SetConfigType("toml")
	err := conf.ReadConfig(&buf)
	if err != nil {
		panic(err)
	}
	return conf
}
