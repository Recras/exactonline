package recras

type Personeel struct {
	ID               int    `json:"id,omitempty"`
	Displaynaam      string `json:"displaynaam"`
	ContactpersoonID int    `json:"contactpersoon_id,omitempty"`
}

func (c *Client) GetCurrentPersoneel() (Personeel, error) {
	p := Personeel{}
	err := c.Get("/api2/personeel/me", &p)
	return p, err
}
