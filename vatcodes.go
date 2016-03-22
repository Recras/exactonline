package exactonline

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Recras/exactonline/httperror"
)

type VATCodeList map[float64]string

const vatCodeURI = "/api/v1/%d/vat/VATCodes"

type VATCode struct {
	ID          string
	Code        string
	Description string
}

type vatCodes struct {
	D struct {
		Results []VATCode `json:"results"`
	} `json:"d"`
}

func (c *Client) GetRecrasVATCodes() ([]VATCode, error) {
	if c.Division == 0 {
		return nil, ErrNoDivision
	}
	filt := url.Values{}
	filt.Set("$filter", "substringof('recras:', Description) eq true")
	u := fmt.Sprintf(vatCodeURI+"?%s", c.Division, filt.Encode())
	resp, err := c.Client.Get(u)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, httperror.New(resp)
	}

	out := &vatCodes{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return nil, err
	}

	return out.D.Results, nil
}
