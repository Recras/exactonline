package exactonline

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/Recras/exactonline/httperror"
)

type Journal struct {
	ID          string
	Code        string
	Description string
}
type journals struct {
	D struct {
		Results []Journal `json:"results"`
	} `json:"d"`
}

const journalURI = "/api/v1/%d/financial/Journals"

var ErrJournalNotFound = errors.New("Journal not found")

func (c *Client) FindDefaultJournal() (Journal, error) {
	if c.Division == 0 {
		return Journal{}, ErrNoDivision
	}
	u := fmt.Sprintf(journalURI, c.Division)
	filt := url.Values{}
	filt.Set("$filter", fmt.Sprintf("Code eq '%s'", "recras"))
	resp, err := c.Client.Get(fmt.Sprintf("%s?%s", u, filt.Encode()))
	if err != nil {
		return Journal{}, err
	}
	if resp.StatusCode != 200 {
		return Journal{}, httperror.New(resp)
	}
	defer resp.Body.Close()

	out := &journals{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return Journal{}, err
	}
	if len(out.D.Results) == 0 {
		return Journal{}, ErrJournalNotFound
	}
	return out.D.Results[0], nil
}
