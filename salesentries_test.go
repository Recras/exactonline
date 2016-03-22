package exactonline

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSalesEntry(t *testing.T) {
	_ = SalesEntry{
		Document: "document-guid",
	}
}

func TestSaveSalesEntry(t *testing.T) {
	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		if r.Method != "POST" {
			t.Errorf("Expected method to be `POST`, got `%s`", r.Method)
		}
		if r.URL.Path != fmt.Sprintf(salesEntriesURI, 123) {
			t.Errorf("Expected path to be set, got %#v", r.URL.Path)
		}
		if h := r.Header.Get("Content-Type"); h != "application/json" {
			t.Errorf("Expected content-type header to be application/json")
		}

		payload := map[string]interface{}{}
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&payload)
		if err != nil {
			panic("Error decoding json, should never happen, error: " + err.Error())
		}

		payload["EntryID"] = "guid"
		payload["EntryDate"] = "/Date(12345)/"
		payload["DueDate"] = "/Date(12345)/"
		payload["SalesEntryLines"] = map[string]interface{}{"__deferred": map[string]interface{}{"uri": "asdf"}}

		out := map[string]interface{}{
			"d": payload,
		}
		enc := json.NewEncoder(w)
		w.WriteHeader(201)
		_ = enc.Encode(&out)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123

	a := SalesEntry{}
	if err := a.Save(cl); err != ErrSalesEntryLinesRequired {
		t.Errorf("Expected ErrSalesEntryLinesRequired if SalesEntryLines is missing, got %#v", err)
	}
	a.SetSalesEntryLines([]SalesEntryLine{})
	if err := a.Save(cl); err != ErrSalesEntryLinesRequired {
		t.Errorf("Expected ErrSalesEntryLinesRequired if SalesEntryLines is empty, got %#v", err)
	}
	a.SetSalesEntryLines([]SalesEntryLine{{}})
	if err := a.Save(cl); err != ErrSalesEntryCustomerRequired {
		t.Errorf("Expected ErrSalesEntryCustomerRequired if Customer is empty, got %#v", err)
	}
	a.Customer = "customer"
	if err := a.Save(cl); err != ErrSalesEntryPaymentConditionRequired {
		t.Errorf("Expecte ErrSalesEntryPaymentConditionRequired, if PaymentCondition is empty, got %#v", err)
	}
	a.PaymentCondition = "paymentcond"

	if err := a.Save(cl); err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if !apiCalled {
		t.Errorf("Expected API to be called")
	}
	if a.ID != "guid" {
		t.Errorf("Expected ID to be set after save") // tijdelijk uitgecommentaard, met andere test bezig
	}
}

func TestFindSalesEntry_NotFound(t *testing.T) {
	salesEntryCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/salesentry/SalesEntries" {
			salesEntryCalled = true
			if f := r.URL.Query().Get("$filter"); f != "substringof('1-2-3', Description) eq true" {
				t.Errorf("Expected $filter query to be `substringof('1-2-3', Description) eq true`, got %#v", f)
			}
			fmt.Fprint(w, `{"d":{"results":[]}}`)
		}
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234
	_, err := cl.FindSalesEntry("1-2-3")
	if !salesEntryCalled {
		t.Errorf("Expected SalesEntries endpoint to be called")
	}
	if e, ok := err.(*ErrSalesEntryNotFound); !ok {
		t.Errorf("Expected ErrSalesEntryNotFound, got %#v", err)
	} else if e.Division != 1234 {
		t.Errorf("Expected ErrSalesEntryNotFound.Division to be 1234, got %#v", e.Division)
	} else if e.FactuurNummer != "1-2-3" {
		t.Errorf("Expected ErrSalesEntryNotFound.FactuurNummer to be %#v, got %#v", "1-2-3", e.FactuurNummer)
	}
}

func TestFindSalesEntry_Found(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/salesentry/SalesEntries" {
			if f := r.URL.Query().Get("$filter"); f == "substringof('1-2-3', Description) eq true" {
				fmt.Fprint(w, `{"d":{"results":[{"EntryID":"guid", "Customer":"customerguid", "Description":"Recras factuur: 1-2-3", "DueDate":"/Date(12345)/", "EntryDate":"/Date(12345)/", "Journal":"recras", "PaymentReference": "1-2-3"}]}}`)
			}
		}
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234
	s, err := cl.FindSalesEntry("1-2-3")
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if s.ID != "guid" {
		t.Errorf("Expected ID field of found SalesEntry to be set")
	}
}

func TestFindSalesEntry_Found_deferredSEL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/salesentry/SalesEntries" {
			if f := r.URL.Query().Get("$filter"); f == "substringof('1-2-3', Description) eq true" {
				fmt.Fprint(w, `{"d":{"results":[
				{"EntryID":"guid", "Customer":"customerguid", "Description":"Recras factuur: 1-2-3", "DueDate":"/Date(12345)/", "EntryDate":"/Date(12345)/", "Journal":"recras", "PaymentReference": "1-2-3", "SalesEntryLines": {"__deferred": {"uri": "/api/v1/1234/salesentry/SalesEntries(guid'guid')/SalesEntryLines"}}}
				]}}`)
			}
		}
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234
	s, err := cl.FindSalesEntry("1-2-3")
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(s.SalesEntryLines()) != 0 {
		t.Errorf("Expected length of SalesEntryLines to be 0, got %d", len(s.SalesEntryLines()))
	}
	if s.DeferredSELines.Deferred.URI != "/api/v1/1234/salesentry/SalesEntries(guid'guid')/SalesEntryLines" {
		t.Errorf("Expected deferred url to be %#v, got %#v", "/api/v1/1234/salesentry/SalesEntries(guid'guid')/SalesEntryLines", s.DeferredSELines.Deferred.URI)
	}
}
