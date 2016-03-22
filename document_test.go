package exactonline

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Recras/exactonline/httperror"
	"golang.org/x/oauth2"
)

func TestDocument(t *testing.T) {
	_ = Document{
		ID:      "guid",
		Subject: "asf",
		Type:    10,
		Account: "account-guid",
	}
}

func TestSaveDocument_empty(t *testing.T) {
	d := Document{
		Type:    10,
		Account: "account-guid",
	}
	cl := &Client{Division: 123}

	err := d.Save(cl)
	if err != ErrNoSubject {
		t.Errorf("Expected ErrNoSubject, got %#v", err)
	}

	d = Document{
		Subject: "asf",
		Account: "account-guid",
	}
	err = d.Save(cl)
	if err != ErrNoType {
		t.Errorf("Expected ErrNoType, got %#v", err)
	}

	d = Document{
		Subject: "asf",
		Type:    10,
	}
	err = d.Save(cl)
	if err != ErrNoAccount {
		t.Errorf("Expected ErrNoAccount, got %#v", err)
	}
}

func TestSaveDocument(t *testing.T) {
	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/documents/Documents" {
			apiCalled = true
			if r.Method != "POST" {
				t.Errorf("Expected HTTP Method 'POST', got %s", r.Method)
			}
			if c := r.Header.Get("Content-type"); c != "application/json" {
				t.Errorf("Expected Content-type header to be 'application/json', got %s", c)
			}
			if c := r.Header.Get("Prefer"); c != "return=representation" {
				t.Errorf("Expected Prefer header to be 'return=representation', got %s", c)
			}
			if c := r.Header.Get("Accept"); c != "application/json" {
				t.Errorf("Expected Accept header to be 'application/json', got %s", c)
			}
			b := map[string]interface{}{}
			dec := json.NewDecoder(r.Body)
			err := dec.Decode(&b)
			if err != nil {
				panic("Error decoding json, should not happen. Error: " + err.Error())
			}

			if b["Type"] != float64(10) {
				t.Errorf("Expected Type to be 10, got %#v", b["Type"])
			}
			if b["Subject"] != "Invoice 1-2-3" {
				t.Errorf("Expected Subject, got %#v", b["Subject"])
			}
			b["ID"] = "document-guid"
			ret := map[string]map[string]interface{}{"d": b}
			enc := json.NewEncoder(w)
			w.WriteHeader(201)
			enc.Encode(ret)
		}
	}))
	defer ts.Close()

	d := Document{
		Type:    10,
		Subject: "Invoice 1-2-3",
		Account: "account-guid",
	}
	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(time.Second)})
	cl.Division = 1234
	err := d.Save(cl)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if !apiCalled {
		t.Errorf("Expected Documents API to be called")
	}

	if d.ID != "document-guid" {
		t.Errorf("Expected ID to be set, got %#v", d)
	}
}

func TestSaveDocument_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	d := Document{
		Type:    10,
		Subject: "Invoice 1-2-3",
		Account: "account-guid",
	}
	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(time.Second)})
	cl.Division = 1234
	err := d.Save(cl)
	if _, ok := err.(httperror.HTTPError); !ok {
		t.Errorf("Expected HTTPError, got %#v", err)
	}
}

func TestSaveDocument_NoDivision(t *testing.T) {
	d := Document{
		Type:    10,
		Subject: "Invoice 1-2-3",
	}
	cl := Client{}
	err := d.Save(&cl)
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}

func TestDocumentAttachment(t *testing.T) {
	_ = DocumentAttachment{
		ID:         "guid",
		Attachment: []byte("asf"),
		Document:   "document-guid",
		FileName:   "test.txt",
	}
}

func TestSaveDocumentAttachment_empty(t *testing.T) {
	d := DocumentAttachment{
		Attachment: []byte("hello"),
		Document:   "document-guid",
	}
	cl := &Client{Division: 123}

	err := d.Save(cl)
	if err != ErrNoFileName {
		t.Errorf("Expected ErrNoFileName, got %#v", err)
	}

	d = DocumentAttachment{
		Attachment: []byte("hello"),
		FileName:   "hello.txt",
	}
	err = d.Save(cl)
	if err != ErrNoDocument {
		t.Errorf("Expected ErrNoDocument, got %#v", err)
	}

	d = DocumentAttachment{
		Document: "document-guid",
		FileName: "hello.txt",
	}
	err = d.Save(cl)
	if err != ErrNoAttachment {
		t.Errorf("Expected ErrNoAttachment, got %#v", err)
	}
}

func TestSaveDocumentAttachment(t *testing.T) {
	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/documents/DocumentAttachments" {
			apiCalled = true
			if r.Method != "POST" {
				t.Errorf("Expected HTTP Method 'POST', got %s", r.Method)
			}
			if c := r.Header.Get("Content-type"); c != "application/json" {
				t.Errorf("Expected Content-type header to be 'application/json', got %s", c)
			}
			if c := r.Header.Get("Prefer"); c != "return=representation" {
				t.Errorf("Expected Prefer header to be 'return=representation', got %s", c)
			}
			if c := r.Header.Get("Accept"); c != "application/json" {
				t.Errorf("Expected Accept header to be 'application/json', got %s", c)
			}
			b := map[string]interface{}{}
			dec := json.NewDecoder(r.Body)
			err := dec.Decode(&b)
			if err != nil {
				panic("Error decoding json, should not happen. Error: " + err.Error())
			}

			if b["Attachment"] != "aGVsbG8=" {
				t.Errorf("Expected Type to be %#v, got %#v", "aGVsbG8=", b["Attachment"])
			}
			if b["Document"] != "document-guid" {
				t.Errorf("Expected Document=`document-guid`, got %#v", b["Document"])
			}
			b["ID"] = "attachment-guid"
			ret := map[string]map[string]interface{}{"d": b}
			enc := json.NewEncoder(w)
			w.WriteHeader(201)
			enc.Encode(ret)
		}
	}))
	defer ts.Close()

	d := DocumentAttachment{
		Document:   "document-guid",
		Attachment: []byte("hello"),
		FileName:   "hello.txt",
	}
	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(time.Second)})
	cl.Division = 1234
	err := d.Save(cl)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if !apiCalled {
		t.Errorf("Expected DocumentAttachments API to be called")
	}

	if d.ID != "attachment-guid" {
		t.Errorf("Expected ID to be set, got %#v", d)
	}
}

func TestSaveDocumentAttachment_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	d := DocumentAttachment{
		Document:   "document-guid",
		FileName:   "hello.txt",
		Attachment: []byte("hello"),
	}
	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(time.Second)})
	cl.Division = 1234
	err := d.Save(cl)
	if _, ok := err.(httperror.HTTPError); !ok {
		t.Errorf("Expected HTTPError, got %#v", err)
	}
}

func TestSaveDocumentAttachment_NoDivision(t *testing.T) {
	d := DocumentAttachment{}
	cl := Client{}
	err := d.Save(&cl)
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}

func TestDocumentType(t *testing.T) {
	_ = DocumentType{
		ID:          12,
		Description: "asf",
	}
}

func TestFindDocumentTypeByDescription_NoDivision(t *testing.T) {
	cl := Client{}
	_, err := cl.FindDocumentTypeByDescription("sadf")
	if err != ErrNoDivision {
		t.Errorf("Expected ErrNoDivision, got %#v", err)
	}
}

func TestFindDocumentTypeByDescription_NotFound(t *testing.T) {
	apiCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/1234/documents/DocumentTypes" {
			apiCalled = true
			if f := r.URL.Query().Get("$filter"); f != "Description eq 'asdf'" {
				t.Errorf("Expected call to filter on 'asdf', got %#v", f)
			}
		}
		fmt.Fprint(w, `{"d":{"results":[]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{
		Expiry: time.Now().Add(1 * time.Second),
	})
	cl.Division = 1234

	_, err := cl.FindDocumentTypeByDescription("asdf")
	if !apiCalled {
		t.Errorf("Expected DocumentTypes API to be called")
	}
	if _, ok := err.(ErrDocumentTypeNotFound); !ok {
		t.Errorf("Expected ErrDocumentTypeNotFound, got %#v", err)
	}
}

func TestFindDocumentTypeByDescription_Found(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"d": {"results": [{"ID": 12, "Description": "asdf"}]}}`)
	}))
	defer ts.Close()

	c := Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(time.Second)})
	cl.Division = 1234

	dt, err := cl.FindDocumentTypeByDescription("asdf")
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if dt.ID != 12 {
		t.Errorf("Expected documentType.ID to be 12, got %#v", dt)
	}
}
