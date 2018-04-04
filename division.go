package exactonline

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Recras/exactonline/httperror"
)

type Me struct {
	CurrentDivision int
	FullName        string
}

type me struct {
	D struct {
		Results []Me `json:"results"`
	} `json:"d"`
}

// GetDefaultDivision retrieves and sets the default division on Client c
func (c *Client) GetDefaultDivision() error {
	resp, err := c.Client.Get("/api/v1/current/Me")
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return httperror.New(resp)
	}
	defer resp.Body.Close()

	out := &me{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return err
	}

	c.Division = out.D.Results[0].CurrentDivision
	return nil
}

type Division struct {
	Code      int
	HID       int64 `json:",string"`
	VATNumber string
	Main      bool
	Country   string
}
type divisions struct {
	D struct {
		Results []Division `json:"results"`
	} `json:"d"`
}

var ErrDivisionNotFound = errors.New("exactonline/api: No division found by VAT number")

func (c *Client) findDivisionByVATNumber(vn string, divisionID int) (Division, error) {
	vnq := url.QueryEscape("VATNumber eq '" + vn + "'")

	resp, err := c.Client.Get(fmt.Sprintf("/api/v1/%d/hrm/Divisions?$filter=%s", divisionID, vnq))
	if err != nil {
		return Division{}, err
	}
	if resp.StatusCode != 200 {
		return Division{}, httperror.New(resp)
	}

	out := &divisions{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return Division{}, err
	}

	if len(out.D.Results) == 0 {
		return Division{}, ErrDivisionNotFound
	}

	return out.D.Results[0], nil
}

func (c *Client) SetDivisionByVATNumber(vn string) error {
	if c.Division == 0 {
		return ErrNoDivision
	}

	vn = strings.Replace(vn, ".", "", -1)

	div, err := c.findDivisionByVATNumber(vn, c.Division)
	if err != nil && err != ErrDivisionNotFound {
		return err
	} else if err == nil {
		c.Division = div.Code
		return nil
	}

	div, err = c.findSystemDivisionByVATNumber(vn)
	if err != nil {
		return err
	}
	c.Division = div.Code
	return nil
}

func (c *Client) findSystemDivisionByVATNumber(vn string) (Division, error) {
	resp, err := c.Client.Get(fmt.Sprintf("/api/v1/%d/system/Divisions", c.Division))
	if err != nil {
		return Division{}, err
	}
	if resp.StatusCode != 200 {
		return Division{}, httperror.New(resp)
	}

	out := &divisions{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return Division{}, err
	}

	for _, div := range out.D.Results {
		d, err := c.findDivisionByVATNumber(vn, div.Code)
		if err != nil && err != ErrDivisionNotFound {
			return Division{}, err
		} else if err == nil {
			return d, nil
		}
	}
	return Division{}, ErrDivisionNotFound
}
