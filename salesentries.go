package exactonline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/Recras/exactonline/httperror"
	"github.com/Recras/exactonline/odata2json"
)

const (
	salesEntriesURI    = "/api/v1/%d/salesentry/SalesEntries"
	salesEntryLinesURI = "/api/v1/%d/salesentry/SalesEntryLines"
)

type SalesEntry struct {
	ID               string `json:"EntryID,omitempty"`
	Customer         string
	Description      string
	DueDate          odata2json.Date
	EntryDate        odata2json.Date
	Journal          string
	PaymentCondition string
	PaymentReference string `json:",omitempty"`
	YourRef          string
	Document         string                  `json:",omitempty"`
	DeferredSELines  deferredSalesEntryLines `json:"SalesEntryLines"`
	Type             int32                   `json:",omitempty"`
}

const (
	SalesEntryNote       = 20
	SalesEntryCreditNote = 21
)

type salesEntries struct {
	D struct {
		Results []SalesEntry `json:"results"`
	} `json:"d"`
}

type deferredSalesEntryLines struct {
	Deferred struct {
		URI string `json:"uri"`
	} `json:"__deferred"`
	SalesEntryLines []SalesEntryLine `json:"results,omitempty"`
}

func (d deferredSalesEntryLines) MarshalJSON() ([]byte, error) {
	bs, err := json.Marshal(d.SalesEntryLines)
	return bs, err
}

func (se SalesEntry) SalesEntryLines() []SalesEntryLine {
	return se.DeferredSELines.SalesEntryLines
}

func (se *SalesEntry) SetSalesEntryLines(lines []SalesEntryLine) {
	se.DeferredSELines.SalesEntryLines = lines
}

type SalesEntryLine struct {
	ID          string `json:",omitempty"`
	AmountFC    float64
	Description string
	EntryID     string `json:",omitempty"`
	GLAccount   string
	Quantity    float64
	VATCode     string
}

type ErrSalesEntryNotFound struct {
	Division      int
	FactuurNummer string
}

func (e ErrSalesEntryNotFound) Error() string {
	return fmt.Sprintf("SalesEntry not found for FactuurNummer `%s` in Division %d", e.Division, e.FactuurNummer)
}

func (c *Client) FindSalesEntry(factuurnr string) (SalesEntry, error) {
	filt := url.Values{}
	filt.Add("$filter", "substringof('"+factuurnr+"', Description) eq true")
	u := fmt.Sprintf(salesEntriesURI, c.Division)
	resp, err := c.Client.Get(u + "?" + filt.Encode())
	if err != nil {
		return SalesEntry{}, err
	}
	if resp.StatusCode != 200 {
		return SalesEntry{}, httperror.New(resp)
	}

	out := &salesEntries{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return SalesEntry{}, err
	}

	if len(out.D.Results) == 0 {
		return SalesEntry{}, &ErrSalesEntryNotFound{c.Division, factuurnr}
	}

	return out.D.Results[0], nil
}

var (
	ErrSalesEntryLinesRequired            = errors.New("Field SalesEntryLines is required on SalesEntry")
	ErrSalesEntryCustomerRequired         = errors.New("Field Customer is required on SalesEntry")
	ErrSalesEntryPaymentConditionRequired = errors.New("Field PaymentCondition is required on SalesEntry")
)

func (s *SalesEntry) Save(ecl *Client) error {
	if s.SalesEntryLines() == nil || len(s.SalesEntryLines()) == 0 {
		return ErrSalesEntryLinesRequired
	}
	if s.Customer == "" {
		return ErrSalesEntryCustomerRequired
	}
	if s.PaymentCondition == "" {
		return ErrSalesEntryPaymentConditionRequired
	}

	bs, err := json.Marshal(s)
	if err != nil {
		return err
	}
	bb := bytes.NewBuffer(bs)
	resp, err := ecl.Client.Post(fmt.Sprintf(salesEntriesURI, ecl.Division), "application/json", bb)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return httperror.New(resp)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	envelope := map[string]SalesEntry{}
	if err := dec.Decode(&envelope); err != nil {
		return err
	}

	*s = envelope["d"]

	return nil
}
