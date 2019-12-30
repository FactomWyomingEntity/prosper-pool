package polling_test

import (
	"net/http"
	"testing"

	. "github.com/FactomWyomingEntity/prosper-pool/polling"
)

// FixedDataSourceTest will test the parsing of the data source using the fixed response
func FixedDataSourceTest(t *testing.T, source string, fixed []byte) {
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

	testDataSource(t, s)
}

// ActualDataSourceTest actually fetches the resp over the internet
func ActualDataSourceTest(t *testing.T, source string) {
	defer func() { http.DefaultClient = &http.Client{} }() // Don't leave http broken

	c := GetConfig("")
	http.DefaultClient = &http.Client{}

	s, err := NewDataSource(source, c)
	if err != nil {
		t.Error(err)
	}

	testDataSource(t, s)
}

func testDataSource(t *testing.T, s IDataSource) {
	pegs, err := s.FetchPegPrices()
	if err != nil {
		t.Error(err)
	}

	for _, asset := range s.SupportedPegs() {
		r, ok := pegs[asset]
		if !ok {
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
