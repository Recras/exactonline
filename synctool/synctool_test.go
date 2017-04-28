package synctool

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/Recras/exactonline"
	"github.com/Recras/exactonline/recras"
	"golang.org/x/oauth2"
)

type itemFinder exactonline.Item

var default_itemfinder = itemFinder{
	ID:          "asdfasdfsadf",
	Code:        "recras12",
	Description: "Cool Recras product",
	IsSalesItem: true,
	Unit:        "recras",
	GLRevenue:   "glaccount",
}

func (i *itemFinder) FindItemByRecrasID(recrasID int) (exactonline.Item, error) {
	return exactonline.Item(*i), nil
}

func Test_convertFactuurregel_Item(t *testing.T) {
	lines, err := convertFactuurregels(&default_itemfinder, []recras.Factuurregel{}, 1, exactonline.VATCodeList{})
	if err != nil {
		t.Errorf("Expected no error when converting empty slice")
	}
	if !reflect.DeepEqual(lines, []exactonline.SalesEntryLine{}) {
		t.Errorf("Expected [], got %#v", lines)
	}

	lines, err = convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:               recras.FactuurregelItem,
		Kortingspercentage: 0,
		Aantal:             2,
		Bedrag:             5,
		ProductID:          2,
		Naam:               "Product",
		BTWPercentage:      21,
	}}, 1, exactonline.VATCodeList{
		21: "R21",
	})
	if err != nil {
		t.Errorf("Expected no error when converting single item")
	}
	if len(lines) != 1 {
		t.Errorf("Expected # of SalesEntryLines to be 1, got %d", len(lines))
	} else {
		if lines[0].AmountFC != 2*5 {
			t.Errorf("Expected amount to be 10, got %f", lines[0].AmountFC)
		}
		if lines[0].GLAccount != "glaccount" {
			t.Errorf("Expected GLAccount to be `glaccount`, got %#v", lines[0].GLAccount)
		}
		if lines[0].Description != "Product" {
			t.Errorf("Expected Description to be `Product`, got %#v", lines[0].Description)
		}
		if lines[0].Quantity != 2 {
			t.Errorf("Expected Quantity to be 2, got %f", lines[0].Quantity)
		}
		if lines[0].VATCode != "R21" {
			t.Errorf("Expected VATCode to be %#v, got %#v", "R21", lines[0].VATCode)
		}
	}
}

func Test_convertFactuurregel_NoVATCode(t *testing.T) {
	_, err := convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:               recras.FactuurregelItem,
		Kortingspercentage: 0,
		Aantal:             2,
		Bedrag:             5,
		ProductID:          2,
		Naam:               "Product",
		BTWPercentage:      21,
	}}, 1, exactonline.VATCodeList{
		6: "R21",
	})
	if _, ok := err.(ErrNoVATCode); !ok {
		t.Errorf("Expected ErrNoVATCode, got %#v", err)
	}
}

func Test_convertFactuurregel_ZeroItem(t *testing.T) {
	vatcodes := exactonline.VATCodeList{
		21: "R21",
	}
	lines, err := convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:               recras.FactuurregelItem,
		Kortingspercentage: 0,
		Aantal:             0,
		Bedrag:             5,
		ProductID:          2,
		Naam:               "Product",
		BTWPercentage:      21,
	}}, 1, vatcodes)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(lines) != 0 {
		t.Errorf("Expected no lines with empty Aantal")
	}

	lines, err = convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:               recras.FactuurregelItem,
		Kortingspercentage: 0,
		Aantal:             1,
		Bedrag:             0,
		ProductID:          2,
		Naam:               "Product",
		BTWPercentage:      21,
	}}, 1, vatcodes)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(lines) != 0 {
		t.Errorf("Expected no lines with empty Bedrag")
	}

	lines, err = convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:               recras.FactuurregelItem,
		Kortingspercentage: 100,
		Aantal:             1,
		Bedrag:             10,
		ProductID:          2,
		Naam:               "Product",
		BTWPercentage:      21,
	}}, 1, vatcodes)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(lines) != 0 {
		t.Errorf("Expected no lines with full reduction")
	}

	lines, err = convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:               recras.FactuurregelItem,
		Kortingspercentage: 0,
		Aantal:             1,
		Bedrag:             10,
		ProductID:          2,
		Naam:               "Product",
		BTWPercentage:      21,
	}}, 0, vatcodes)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(lines) != 0 {
		t.Errorf("Expected no lines with kortingsfactor 0")
	}

	itemf := itemFinder{
		ID:        "asdf",
		Code:      "recras12",
		GLRevenue: "",
	}
	lines, err = convertFactuurregels(&itemf, []recras.Factuurregel{
		{
			Type:               recras.FactuurregelItem,
			ProductID:          12,
			Kortingspercentage: 0,
			Aantal:             1,
			Bedrag:             0,
		},
	}, 1, exactonline.VATCodeList{})
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(lines) != 0 {
		t.Errorf("Expected no with empty Bedrag and no GLRevenue")
	}
}

func Test_convertFactuurregel_noGLRevenue(t *testing.T) {
	itemf := itemFinder{
		ID:        "asdf",
		Code:      "recras12",
		GLRevenue: "",
	}
	_, err := convertFactuurregels(&itemf, []recras.Factuurregel{
		{
			Type:               recras.FactuurregelItem,
			ProductID:          12,
			Kortingspercentage: 0,
			Aantal:             1,
			Bedrag:             10,
		},
	}, 1, exactonline.VATCodeList{})
	if e, ok := err.(ErrNoGLRevenueAccount); !ok {
		t.Errorf("Expected ErrNoGLRevenueAccount, got %#v", err)
	} else if e.ProductID != 12 {
		t.Errorf("Expected error.ProductID to be %#v, got %#v", 12, e.ProductID)
	}
}

func Test_convertFactuurregel_MultipleItems(t *testing.T) {
	vatcodes := exactonline.VATCodeList{
		6:  "R6",
		21: "R21",
	}
	lines, err := convertFactuurregels(&default_itemfinder, []recras.Factuurregel{
		{
			Type:               recras.FactuurregelItem,
			Kortingspercentage: 0,
			Aantal:             2,
			Bedrag:             5,
			ProductID:          2,
			Naam:               "Product2",
			BTWPercentage:      21,
		}, {
			Type:               recras.FactuurregelItem,
			Kortingspercentage: 0,
			Aantal:             4,
			Bedrag:             2,
			ProductID:          4,
			Naam:               "Product4",
			BTWPercentage:      6,
		}}, 1, vatcodes)
	if err != nil {
		t.Errorf("Expected no error when converting single item")
	}
	if len(lines) != 2 {
		t.Errorf("Expected # of SalesEntryLines to be 2, got %d", len(lines))
	}

	lines, err = convertFactuurregels(&default_itemfinder, []recras.Factuurregel{
		{
			Type:               recras.FactuurregelItem,
			Kortingspercentage: 0,
			Aantal:             2,
			Bedrag:             0,
			ProductID:          12,
			Naam:               "Product12",
			BTWPercentage:      21,
		}, {
			Type:               recras.FactuurregelItem,
			Kortingspercentage: 0,
			Aantal:             4,
			Bedrag:             2,
			ProductID:          4,
			Naam:               "Product4",
			BTWPercentage:      6,
		}}, 1, vatcodes)
	if err != nil {
		t.Errorf("Expected no error when converting multiple item")
	}
	if len(lines) != 1 {
		t.Errorf("Expected # of SalesEntryLines to be 1, got %d", len(lines))
	}
}

func Test_convertFactuurregel_NegativeItem(t *testing.T) {
	vatcodes := exactonline.VATCodeList{
		6: "R6",
	}
	lines, err := convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:          recras.FactuurregelItem,
		ProductID:     12,
		Bedrag:        -10,
		Aantal:        1,
		BTWPercentage: 6,
	}}, 1, vatcodes)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(lines) != 1 {
		t.Errorf("Expected 1 lines, got %d", len(lines))
	} else if lines[0].AmountFC != -10 {
		t.Errorf("Expected AmountFC to be %#v, got %#v", -10, lines[0].AmountFC)
	}
}

func Test_convertFactuurregel_Group(t *testing.T) {
	vatcodes := exactonline.VATCodeList{
		6: "R6",
	}
	lines, err := convertFactuurregels(&default_itemfinder, []recras.Factuurregel{{
		Type:               recras.FactuurregelGroep,
		Kortingspercentage: 10,
		Regels: []recras.Factuurregel{{
			Type:          recras.FactuurregelItem,
			Aantal:        2,
			Bedrag:        5,
			ProductID:     4,
			Naam:          "Product4",
			BTWPercentage: 6,
		}},
	}}, 1, vatcodes)
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}
	if len(lines) != 1 {
		t.Errorf("Expected 1 line")
	} else {
		if lines[0].AmountFC != 9 {
			t.Errorf("Expected Kortingspercentage on Groep to weigh in on Item, got amount %f", lines[0].AmountFC)
		}
	}
}

func Test_convertFactuur_empty(t *testing.T) {
	_ = convertFactuur(&default_itemfinder, exactonline.Account{}, recras.Factuur{}, exactonline.PaymentCondition{})
}

func Test_convertFactuur_simple(t *testing.T) {
	ac := exactonline.Account{
		ID:   "account-guid",
		Name: "Test_convertFactuur",
	}
	factuur := recras.Factuur{
		FactuurNummer:   "1-2-3",
		Datum:           recras.Date{time.Date(2014, 7, 11, 0, 0, 0, 0, time.UTC)},
		Betaaltermijn:   14,
		ReferentieKlant: "asdf-123",
	}
	se := convertFactuur(&default_itemfinder, ac, factuur, exactonline.PaymentCondition{Code: "pcCode"})
	if se.PaymentReference != factuur.FactuurNummer {
		t.Errorf("Expected PaymentReference to be %#v, got %#v", factuur.FactuurNummer, se.PaymentReference)
	}
	if se.Description != "Recras factuur: "+factuur.FactuurNummer {
		t.Errorf("Expected Description to be %#v, got %#v", "Recras factuur: "+factuur.FactuurNummer, se.Description)
	}
	if se.EntryDate.Time != factuur.Datum.Time {
		t.Errorf("Expected EntryDate to be %#s, got %#s", factuur.Datum, se.EntryDate.Time)
	}
	betaaldatum := factuur.Datum.Time.AddDate(0, 0, factuur.Betaaltermijn)
	if se.DueDate.Time != betaaldatum {
		t.Errorf("Expected DueDate to be %#s, got %#s", betaaldatum, se.DueDate.Time)
	}
	if se.Journal != "recras" {
		t.Errorf("Expected Journal to be %#v, got %#v", "recras", se.Journal)
	}
	if se.Customer != ac.ID {
		t.Errorf("Expected Customer to be %#v, got %#v", ac.ID, se.Customer)
	}
	if se.YourRef != factuur.ReferentieKlant {
		t.Errorf("Expected YourRef to be %#v, got %#v", factuur.ReferentieKlant, se.YourRef)
	}
	if se.PaymentCondition != "pcCode" {
		t.Errorf("Expected PaymentCondition to be %#v, got %#v", "pcCode", se.PaymentCondition)
	}
	if se.SalesEntryLines() != nil {
		t.Errorf("Expected SalesEntryLines to be %#v, got %#v", nil, se.SalesEntryLines())
	}
}

func Test_convertFactuur_negative(t *testing.T) {
	ac := exactonline.Account{}
	factuur := recras.Factuur{
		FactuurNummer:                      "1-2-3",
		CalculatedTotaalbedragInclusiefBTW: -10,
		Datum: recras.Date{time.Date(2014, 7, 11, 0, 0, 0, 0, time.UTC)},
	}
	se := convertFactuur(&default_itemfinder, ac, factuur, exactonline.PaymentCondition{})
	if se.Type != exactonline.SalesEntryCreditNote {
		t.Errorf("Expected Type to be %d, got %d", exactonline.SalesEntryCreditNote, se.Type)
	}
}

func Test_uploadFactuurPDF_full(t *testing.T) {
	factuurDownloaded := false
	documentTypeAPICalled := false
	documentsAPICalled := false
	documentAttachmentsAPICalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/facturen/factuur.pdf" {
			factuurDownloaded = true
			fmt.Fprint(w, "Hello, world!")
		}
		if r.URL.Path == "/api/v1/123/documents/DocumentTypes" {
			documentTypeAPICalled = true
			if f := r.URL.Query().Get("$filter"); f != "Description eq 'Sales invoice'" {
				t.Errorf("Expected DocumentTypes API to be queried for 'Sales invoice', got %#v", f)
			}
			fmt.Fprint(w, `{"d": {"results": [{"ID": 10}]}}`)
		}
		if r.URL.Path == "/api/v1/123/documents/Documents" {
			documentsAPICalled = true
			var pl map[string]interface{}
			dec := json.NewDecoder(r.Body)
			dec.Decode(&pl)
			pl["ID"] = "document-guid"
			ret := map[string]map[string]interface{}{"d": pl}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(ret)
		}
		if r.URL.Path == "/api/v1/123/documents/DocumentAttachments" {
			documentAttachmentsAPICalled = true
			var pl map[string]interface{}
			dec := json.NewDecoder(r.Body)
			dec.Decode(&pl)

			if pl["Document"] != "document-guid" {
				t.Errorf("Expected Attachment.Document to be `document-guid`, got %#v", pl["Document"])
			}
			if pl["FileName"] != "factuur.pdf" {
				t.Errorf("Expected Attachment.FileName to be `factuur.pdf`, got %#v", pl["FileName"])
			}
			if pl["Attachment"] != "SGVsbG8sIHdvcmxkIQ==" {
				t.Errorf("Expected Attachment.Attachment to be `SGVsbG8sIHdvcmxkIQ==`, got %#v", pl["Attachment"])
			}

			pl["ID"] = "attachment-guid"
			pl["Attachment"] = nil
			ret := map[string]map[string]interface{}{"d": pl}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(ret)
		}
	}))
	defer ts.Close()

	c := exactonline.Config{BaseURL: ts.URL}
	cl := c.NewClient(oauth2.Token{Expiry: time.Now().Add(3 * time.Second)})
	cl.Division = 123

	r := recras.NewClient("test.recras.nl", "username", "password")
	base, _ := url.Parse(ts.URL)
	r.Client.Transport = &recras.Transport{BaseURL: base}

	f := recras.Factuur{
		FactuurNummer: "1-2-3",
		PdfLocatie:    "factuur.pdf",
		KlantID:       120,
	}
	a := exactonline.Account{ID: "account-guid"}
	doc, err := uploadFactuurPDF(&r, cl, f, a)
	if !factuurDownloaded {
		t.Errorf("Expected factuur.pdf to be downloaded")
	}
	if !documentTypeAPICalled {
		t.Errorf("Expected exact DocumentType API to be called")
	}
	if !documentsAPICalled {
		t.Errorf("Expected exact documents API to be called")
	}
	if !documentAttachmentsAPICalled {
		t.Errorf("Expected exact documentAttachments API to be called")
	}
	if err != nil {
		t.Errorf("Expected no error, got %#v", err)
	}

	if doc == nil {
		t.Errorf("Expected document to be set, got nil")
	} else {
		if doc.Type != 10 {
			t.Errorf("Expected document type to be 10, got %#v", doc.Type)
		}
		if doc.Account != "account-guid" {
			t.Errorf("Expected document account to be `account-guid`, got %#v", doc.Account)
		}
		if doc.Subject != "Recras factuur 1-2-3" {
			t.Errorf("Expected document subject to be `Recras factuur 1-2-3`, got %#v", doc.Subject)
		}
		if doc.ID != "document-guid" {
			t.Errorf("Expected document ID to be `document-guid`, got %#v", doc.ID)
		}
	}
}

func Test_uploadFactuurPDF_fileNotFound(t *testing.T) {
}
