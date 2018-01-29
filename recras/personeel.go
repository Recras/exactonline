package recras

import (
	"strconv"
)

type Personeel struct {
	ID               int    `json:"id,omitempty"`
	Displaynaam      string `json:"displaynaam"`
	ContactpersoonID int    `json:"contactpersoon_id,omitempty"`
}

type Rol struct {
	ID int `json:"id,omitempty"`
}

type Gebruiker struct {
	ID     int   `json:"id,omitempty"`
	Rollen []Rol `json:"rollen,omitempty"`
}

func (c *Client) GetCurrentPersoneel() (Personeel, error) {
	p := Personeel{}
	err := c.Get("/api2/personeel/me", &p)
	return p, err
}

func (c *Client) GetGebruiker(p Personeel) (Gebruiker, error) {
	g := Gebruiker{}
	err := c.Get("/api2/gebruikers/"+strconv.Itoa(p.ID)+"?embed=rollen", &g)
	return g, err
}

func (g *Gebruiker) GetFirstRolId() int {
	var rolId int
	if len(g.Rollen) > 0 {
		rolId = g.Rollen[0].ID
	}
	return rolId
}
