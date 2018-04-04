package exactonline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestGetDefaultDivision(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/current/Me" {
			t.Errorf("Expected URL path to be `/api/v1/current/Me`, got `%s`", r.URL.Path)
		}
		if r.Header.Get("accept") != "application/json" {
			t.Errorf("Expected request to accept application/json, got %#v", r.Header["accept"])
		}

		fmt.Fprint(w, `{"d":{"results":[{"__metadata":{"url":"asf"},"CurrentDivision":1234,"FullName":"Unit Test"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(3 * time.Second)})
	err := cl.GetDefaultDivision()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	} else if cl.Division != 1234 {
		t.Errorf("Expected currentDivision to be 1234, got %d", cl.Division)
	}
}

func TestSetDivisionWithoutDivision(t *testing.T) {
	c := Config{}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(3 * time.Second)})
	err := cl.SetDivisionByVATNumber("")
	if err != ErrNoDivision {
		t.Errorf("Expected error to be ErrNoDivision")
	}
}

func TestSetDivisionCredentials(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer valid" {
			http.Error(w, "Unauthorized", 401)
			return
		}
		if r.URL.Path != "/api/v1/1234/hrm/Divisions" {
			t.Errorf("Expected URL Path to be `/api/v1/1234/hrm/Divisions`, got %#v", r.URL.Path)
		}
		if vn := r.URL.Query()["$filter"]; vn[0] != "VATNumber eq 'NL123456789B01'" {
			t.Errorf("Expected $filter query parameter to be VAT number, got %#v", vn)
		}

		fmt.Fprintf(w, `{"d":{"results":[{"Code": 456, "HID": "789", "VATNumber": "NL123456789B01"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		AccessToken: "invalid",
		Expiry:      time.Now().Add(3 * time.Second),
	})
	cl.Division = 1234
	err := cl.SetDivisionByVATNumber("NL123456789B01")

	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	cl = c.NewClient(oauth2.Token{
		AccessToken: "valid",
		Expiry:      time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234

	err = cl.SetDivisionByVATNumber("NL123456789B01")
	if err != nil {
		t.Errorf("expected no error, got %#v", err)
	}
	if cl.Division != 456 {
		t.Errorf("expected division with code 456, got %#v", cl.Division)
	}
}

func TestSetDivisionPeriods(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if vn := r.URL.Query()["$filter"]; vn[0] != "VATNumber eq 'NL123456789B01'" {
			t.Errorf("Expected $filter query parameter to to have stripped periods, got %#v", vn)
		}

		fmt.Fprintf(w, `{"d":{"results":[{"Code": 456, "HID": "789", "VATNumber": "NL123456789B01"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234

	cl.SetDivisionByVATNumber("NL12.3456.789.B01")
}

func TestSetDivisionSystemDivisionsSingleResult(t *testing.T) {
	systemDivisionsCalled := false
	division1235Called := false
	division1236Called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/hrm/Divisions" {
			fmt.Fprintf(w, `{"d":{"results":[]}}`)
			return
		}
		if r.URL.Path == "/api/v1/1234/system/Divisions" {
			systemDivisionsCalled = true
			fmt.Fprintf(w, `{"d":{"results":[{"Code":1236},{"Code": 1235}]}}`)
			return
		}
		if r.URL.Path == "/api/v1/1236/hrm/Divisions" {
			division1236Called = true
			fmt.Fprintf(w, `{"d":{"results":[]}}`)
			return
		}
		if r.URL.Path == "/api/v1/1235/hrm/Divisions" {
			division1235Called = true
			if vn := r.URL.Query()["$filter"]; vn[0] != "VATNumber eq 'NL123456789B01'" {
				t.Errorf("Expected $filter query parameter to be VAT number, got %#v", vn)
			}
			fmt.Fprintf(w, `{"d":{"results":[{"Code": 456, "HID": "789", "VATNumber": "NL123456789B01"}]}}`)
			return
		}
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		AccessToken: "valid",
		Expiry:      time.Now().Add(3 * time.Second),
	})
	cl.Division = 1234
	err := cl.SetDivisionByVATNumber("NL123456789B01")
	if !systemDivisionsCalled {
		t.Errorf("Expected SetDivisionByVATNumber to call system/Divisions API when hrm/Division does have results")
	}
	if !division1235Called {
		t.Errorf("Expected SetDivisionByVATNumber to call other hrm/Divisions API when hrm/Division does have results")
	}
	if !division1236Called {
		t.Errorf("Expected SetDivisionByVATNumber to call other hrm/Divisions API when hrm/Division does have results")
	}
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if cl.Division != 456 {
		t.Errorf("Expected Division with code 456, got %#v", cl.Division)
	}
}
