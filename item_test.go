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

func TestGetAllItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/123/logistics/Items" {
			t.Errorf("Expected path to be correct, got %#v", r.URL.Path)
		}

		fmt.Fprint(w, `{"d":{"results":[{"ID":"guid1","Code":"recras1"},{"ID":"guid2","Code":"recras2"}]}}`)
	}))
	defer ts.Close()
	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	items, err := cl.GetAllItems()
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	} else if (items[0] != Item{ID: "guid1", Code: "recras1"}) {
		t.Errorf("Expected first item, got %#v", items[0])
	}
}

func TestGetAllItems_NoDivision(t *testing.T) {
	cl := Client{}
	_, err := cl.GetAllItems()
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}

func TestFindItemByRecrasID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; p != "/api/v1/123/logistics/Items" {
			t.Errorf("Expected path to be `/api/v1/123/logistics/Items`, got `%s`", p)
		}
		if f := r.URL.Query().Get("$filter"); f != "Code eq 'recras12'" {
			t.Errorf("Expected $filter to be `Code eq 'recras12'`, got `%s`", f)
		}
		fmt.Fprint(w, `{"d":{"results":[{"Code": "recras12", "ID": "guid"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	item, err := cl.FindItemByRecrasID(12)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if (item != Item{Code: "recras12", ID: "guid"}) {
		t.Errorf("Expected the item, got %#v", item)
	}
}
func TestFindItemByRecrasID_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; p != "/api/v1/123/logistics/Items" {
			t.Errorf("Expected path to be `/api/v1/123/logistics/Items`, got `%s`", p)
		}
		if f := r.URL.Query().Get("$filter"); f != "Code eq 'recras12'" {
			t.Errorf("Expected $filter to be `Code eq 'recras12'`, got `%s`", f)
		}
		fmt.Fprint(w, `{"d":{"results":[]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	_, err := cl.FindItemByRecrasID(12)
	if err == nil {
		t.Errorf("Expected error")
	} else if _, ok := err.(ErrItemNotFound); !ok {
		t.Errorf("Expected ErrItemNotFound, got %#v", err)
	}
}

func TestFindItemByRecrasID_NoDivision(t *testing.T) {
	cl := Client{}
	_, err := cl.FindItemByRecrasID(12)
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}

func TestSaveItem_NoDivision(t *testing.T) {
	cl := &Client{}
	i := Item{}
	err := i.Save(cl)
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}

func TestSaveItem(t *testing.T) {
	apicalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apicalled = true
		if r.Method != "POST" {
			t.Errorf("Expected method to be `POST`, got `%s`", r.Method)
		}
		if r.URL.Path != fmt.Sprintf(itemURI, 123) {
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

		if payload["Code"] != "code" {
			t.Errorf("Expected Code to be %#v, got %#v", "code", payload["Code"])
		}
		if payload["Description"] != "desc" {
			t.Errorf("Expected Description to be %#v, got %#v", "desc", payload["Desc"])
		}

		w.WriteHeader(201)

		payload["ID"] = "guid"
		payload["StartDate"] = "/Date(1234)/"

		out := map[string]interface{}{
			"d": payload,
		}
		enc := json.NewEncoder(w)
		_ = enc.Encode(&out)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(time.Second),
	})
	cl.Division = 123

	a := Item{}

	if err := a.Save(cl); err != ErrItemCodeRequired {
		t.Errorf("Expected ErrItemCodeRequired, got %#v", err)
	}

	a.Code = "code"
	if err := a.Save(cl); err != ErrItemDescriptionRequired {
		t.Errorf("Expected ErrItemDescriptionRequired, %#v", err)
	}

	a.Description = "desc"
	if err := a.Save(cl); err != ErrItemUnitRequired {
		t.Errorf("Expected ErrItemUnitRequired, got %#v", err)
	}

	a.Unit = "recras"

	a.Save(cl)
	if !apicalled {
		t.Errorf("Expected API to be called")
	}
	if err := a.Save(cl); err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}

	if a.ID != "guid" {
		t.Errorf("Expected ID to be %#v, got %#v", "guid", a.ID)
	}
}

func TestFindDefaultItemGroup(t *testing.T) {
	apicalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apicalled = true

		if p := r.URL.Path; p != "/api/v1/123/logistics/ItemGroups" {
			t.Errorf("Expect path to be `/api/v1/123/logistics/ItemGroups`, got `%s`", p)
		}
		fmt.Fprint(w, `{"d":{"results":[{"ID":"guid","Code":"recras12"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(time.Second),
	})
	cl.Division = 123

	itemgroup, err := cl.FindDefaultItemGroup()
	if !apicalled {
		t.Errorf("Expected API to be called")
	}
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if (itemgroup != ItemGroup{Code: "recras12", ID: "guid"}) {
		t.Errorf("Expected the item, got %#v", itemgroup)
	}
}

func TestFindDefaultItemGroup_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"d":{"results":[]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 123
	_, err := cl.FindDefaultItemGroup()
	if err != ErrNoDefaultItemGroup {
		t.Errorf("Expected ErrNoDefaultItemGroup, got %#v", err)
	}
}
