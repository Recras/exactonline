package exactonline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/Recras/exactonline/httperror"
)

const accountURI = "/api/v1/%d/crm/Accounts"

type Account struct {
	ID           string `json:",omitempty"`
	Code         string `json:",omitempty"`
	Name         string
	AddressLine1 string `json:",omitempty"`
	Postcode     string `json:",omitempty"`
	City         string `json:",omitempty"`
	SearchCode   string `json:",omitempty"`
	Status       string `json:",omitempty"`
}

type accounts struct {
	D struct {
		Results []Account `json:"results"`
	} `json:"d"`
}

type ErrAccountNotFound struct {
	Division int
	RecrasID int
}

func (e ErrAccountNotFound) Error() string {
	return fmt.Sprintf("Account not found for RecrasID `%s` in Division %d", e.Division, e.RecrasID)
}

func (c *Client) findAccountByFilter(recrasID int, filter string) (Account, error) {
	u := fmt.Sprintf(accountURI, c.Division)
	filt := url.Values{}
	filt.Set("$filter", filter)
	resp, err := c.Client.Get(fmt.Sprintf("%s?%s", u, filt.Encode()))
	if err != nil {
		return Account{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Account{}, httperror.New(resp)
	}

	out := &accounts{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return Account{}, err
	}
	if len(out.D.Results) == 0 {
		return Account{}, ErrAccountNotFound{c.Division, recrasID}
	}

	return out.D.Results[0], nil
}

func (c *Client) FindAccountByRecrasID(recrasID int) (Account, error) {
	if c.Division == 0 {
		return Account{}, ErrNoDivision
	}
	a, err := c.findAccountByFilter(recrasID, fmt.Sprintf("SearchCode eq 'K%d'", recrasID))
	if _, ok := err.(ErrAccountNotFound); ok {
		a, err = c.findAccountByFilter(recrasID, fmt.Sprintf("Code eq '732727%012d'", recrasID))
	}
	return a, err
}

var ErrAccountNameRequired = errors.New("Field `Name` on type `Account` is mandatory")

func (a *Account) Save(ecl *Client) error {
	if ecl.Division == 0 {
		return ErrNoDivision
	}

	if a.Name == "" {
		return ErrAccountNameRequired
	}
	bs, err := json.Marshal(a)
	if err != nil {
		return err
	}
	bb := bytes.NewBuffer(bs)
	resp, err := ecl.Client.Post(fmt.Sprintf(accountURI, ecl.Division), "application/json", bb)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	envelope := map[string]Account{}
	err = dec.Decode(&envelope)
	if err != nil {
		return err
	}

	*a = envelope["d"]
	return nil
}
