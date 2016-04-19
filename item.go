package exactonline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)
import (
	"github.com/Recras/exactonline/httperror"
	"github.com/Recras/exactonline/odata2json"
)

const itemURI = "/api/v1/%d/logistics/Items"

type Item struct {
	ID          string `json:",omitempty"`
	Code        string
	Description string
	StartDate   odata2json.Date
	IsSalesItem bool
	Unit        string `json:",omitempty"`
	GLRevenue   string `json:",omitempty"`
}

type items struct {
	D struct {
		Results []Item `json:"results"`
	} `json:"d"`
}

func (c *Client) GetAllItems() ([]Item, error) {
	if c.Division == 0 {
		return nil, ErrNoDivision
	}
	u := fmt.Sprintf(itemURI, c.Division)
	resp, err := c.Client.Get(u)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, httperror.New(resp)
	}

	out := &items{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return nil, err
	}

	return out.D.Results, nil
}

type ErrItemNotFound struct {
	Division int
	RecrasID int
}

func (e ErrItemNotFound) Error() string {
	return fmt.Sprintf("Item not found for RecrasID `%d` in Division %d", e.RecrasID, e.Division)
}

type ItemFinder interface {
	FindItemByRecrasID(int) (Item, error)
}

func (c *Client) FindItemByRecrasID(recrasID int) (Item, error) {
	if c.Division == 0 {
		return Item{}, ErrNoDivision
	}
	u := fmt.Sprintf(itemURI, c.Division)
	filt := url.Values{}
	filt.Set("$filter", fmt.Sprintf("Code eq 'recras%d'", recrasID))
	resp, err := c.Client.Get(fmt.Sprintf("%s?%s", u, filt.Encode()))
	if err != nil {
		return Item{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Item{}, httperror.New(resp)
	}

	out := &items{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return Item{}, err
	}
	if len(out.D.Results) == 0 {
		return Item{}, ErrItemNotFound{c.Division, recrasID}
	}

	return out.D.Results[0], nil
}

var (
	ErrItemCodeRequired        = errors.New("Field `Code` on type `Item` is mandatory")
	ErrItemDescriptionRequired = errors.New("Field `Description` on type `Item` is mandatory")
	ErrItemUnitRequired        = errors.New("Field `Unit` on type `Item` is mandatory")
)

func (i *Item) Save(c *Client) error {
	if c.Division == 0 {
		return ErrNoDivision
	}
	if i.Code == "" {
		return ErrItemCodeRequired
	}
	if i.Description == "" {
		return ErrItemDescriptionRequired
	}
	if i.Unit == "" {
		return ErrItemUnitRequired
	}

	bs, err := json.Marshal(i)
	if err != nil {
		return err
	}

	bb := bytes.NewBuffer(bs)
	resp, err := c.Client.Post(fmt.Sprintf(itemURI, c.Division), "application/json", bb)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return httperror.New(resp)
	}

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	envelope := map[string]Item{}
	err = dec.Decode(&envelope)
	if err != nil {
		return err
	}

	*i = envelope["d"]

	return nil
}

const itemGroupURI = "/api/v1/%d/logistics/ItemGroups"

type ItemGroup struct {
	ID   string
	Code string
}
type itemgroups struct {
	D struct {
		Results []ItemGroup `json:"results"`
	} `json:"d"`
}

var ErrNoDefaultItemGroup = errors.New("No default ItemGroup")

func (c *Client) FindDefaultItemGroup() (ItemGroup, error) {
	u := fmt.Sprintf(itemGroupURI, c.Division)
	filt := url.Values{}
	filt.Set("$filter", "IsDefault eq 1")
	resp, err := c.Client.Get(fmt.Sprintf("%s?%s", u, filt.Encode()))
	if err != nil {
		return ItemGroup{}, err
	}
	if resp.StatusCode != 200 {
		return ItemGroup{}, httperror.New(resp)
	}

	out := &itemgroups{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(out)
	if err != nil {
		return ItemGroup{}, err
	}

	if len(out.D.Results) == 0 {
		return ItemGroup{}, ErrNoDefaultItemGroup
	}
	return out.D.Results[0], nil
}
