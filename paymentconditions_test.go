package exactonline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestFindPaymentConditionByDescription_notFound(t *testing.T) {
	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalled = true
		if r.URL.Path != "/api/v1/1234/cashflow/PaymentConditions" {
			t.Errorf("Expected url to be %#v, got %#v", `/api/v1/1234/cashflow/PaymentConditions`, r.URL.Path)
		}
		if f := r.URL.Query().Get("$filter"); f != "Description eq 'test'" {
			t.Errorf("Expected $filter Description == 'test', got %#v", f)
		}
		fmt.Fprint(w, `{"d": {"results": []}}`)
	}))
	defer ts.Close()
	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(time.Second),
	})
	cl.Division = 1234
	_, err := cl.FindPaymentConditionByDescription("test")
	if err != ErrPaymentConditionNotFound {
		t.Errorf("Expected ErrPaymentConditionNotFound, got %#v", err)
	}
	if !apiCalled {
		t.Errorf("Expected API to be called")
	}
}

func TestFindPaymentConditionByDescription_found(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/1234/cashflow/PaymentConditions" {
			t.Errorf("Expected url to be %#v, got %#v", `/api/v1/1234/cashflow/PaymentConditions`, r.URL.Path)
		}
		if f := r.URL.Query().Get("$filter"); f != "Description eq 'test'" {
			t.Errorf("Expected $filter Description == 'test', got %#v", f)
		}
		fmt.Fprint(w, `{"d": {"results": [{"ID": "paymentcondition-guid", "Code": "re", "Description": "test"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(time.Second),
	})
	cl.Division = 1234
	pc, err := cl.FindPaymentConditionByDescription("test")
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if pc.ID != "paymentcondition-guid" {
		t.Errorf("Expected ID to be %#v, got %#v", "paymentcondition-guid", pc.ID)
	}
}
