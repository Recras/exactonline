package recras

import "net/url"

type Bedrijf struct {
	ID           int    `json:"id"`
	Bedrijfsnaam string `json:"bedrijfsnaam"`
	BTWNummer    string `json:"btw_nummer"`
}

func (c *Client) GetBedrijven(f url.Values) ([]Bedrijf, error) {
	if f == nil {
		f = url.Values{}
	}
	url := "/api2/bedrijven?" + f.Encode()
	out := []Bedrijf{}
	err := c.Get(url, &out)
	return out, err
}
