package recras

import (
	"errors"
	"time"
)

type Contactmoment struct {
	ID    int    `json:"id,omitempty"`
	Soort string `json:"soort_contact"`

	ContactID        int `json:"contact_id"`
	ContactpersoonID int `json:"contactpersoon_id"`

	Onderwerp     string `json:"onderwerp"`
	Bericht       string `json:"bericht"`
	Ondertekening string `json:"ondertekening"`

	Sticky bool `json:"sticky"`

	ContactOpnemen          *time.Time `json:"contact_opnemen"`
	ContactOpnemenGroup     int        `json:"contact_opnemen_group"`
	ContactOpnemenOpmerking string     `json:"contact_opnemen_opmerking"`
}

var (
	ErrNoSoort = errors.New("recras.Contactmoment.Soort cannot be empty")
)

func (c *Contactmoment) Save(client *Client) error {
	if c.Soort == "" {
		return ErrNoSoort
	}
	return client.Post("/api2/contactmomenten", c)
}
