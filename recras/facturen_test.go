package recras

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestGetAllFacturenWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id": 1, "klant_id": 2, "status": "verzonden", "factuur_nummer": "2-4-1", "datum": "2015-01-01", "betaaltermijn": 14, "bedrijf_id": 3, "regels": []}]`)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	c := &Client{Client: http.Client{
		Transport: &Transport{
			BaseURL: u,
		},
	}}

	fs, err := c.GetFacturenFilter(nil)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(fs) != 1 {
		t.Errorf("Expected number of Facturen to be 1, got %d", len(fs))
	}
	compare := Factuur{
		ID:            1,
		KlantID:       2,
		Status:        "verzonden",
		FactuurNummer: "2-4-1",
		Datum:         Date{time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)},
		Betaaltermijn: 14,
		BedrijfID:     3,
		Regels:        []Factuurregel{},
	}
	if !reflect.DeepEqual(fs[0], compare) {
		t.Errorf("Expected %v, got %v", compare, fs[0])
	}
}
