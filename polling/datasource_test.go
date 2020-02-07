package polling_test

import (
	"net/http"
	"testing"

	. "github.com/FactomWyomingEntity/prosper-pool/polling"
)

// FixedDataSourceTest will test the parsing of the data source using the fixed response
func FixedDataSourceTest(t *testing.T, source string, fixed []byte, exceptions ...string) {
	defer func() { http.DefaultClient = &http.Client{} }() // Don't leave http broken

	c := GetConfig("")

	// Set default http client to return what we expect from apilayer
	cl := GetClientWithFixedResp(fixed)
	http.DefaultClient = cl
	NewHTTPClient = func() *http.Client {
		return GetClientWithFixedResp(fixed)
	}

	s, err := NewDataSource(source, c)
	if err != nil {
		t.Error(err)
	}

	testDataSource(t, s, exceptions...)
}

// ActualDataSourceTest actually fetches the resp over the internet
func ActualDataSourceTest(t *testing.T, source string, exceptions ...string) {
	defer func() { http.DefaultClient = &http.Client{} }() // Don't leave http broken

	c := GetConfig("")

	NewHTTPClient = func() *http.Client {
		return &http.Client{}
	}
	http.DefaultClient = &http.Client{}

	s, err := NewDataSource(source, c)
	if err != nil {
		t.Error(err)
	}

	testDataSource(t, s, exceptions...)
}

func testDataSource(t *testing.T, s IDataSource, exceptions ...string) {
	pegs, err := s.FetchPegPrices()
	if err != nil {
		t.Error(err)
	}

	exceptionsMap := make(map[string]interface{})
	for _, e := range exceptions {
		exceptionsMap[e] = struct{}{}
	}

	for _, asset := range s.SupportedPegs() {
		r, ok := pegs[asset]
		if !ok {
			if _, except := exceptionsMap[asset]; except {
				continue // This asset is an exception from the check
			}
			t.Errorf("Missing %s", asset)
		}

		err := PriceCheck(asset, r.Value)
		if err != nil {
			t.Error(err)
		}

		if r.When.Year() < 2010 || r.When.Year() > 2050 {
			t.Errorf("Year is incorrect: %d", r.When.Year())
		}
	}
}
