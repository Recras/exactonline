package recras

type Product struct {
	ID            int    `json:"id"`
	LeverancierID int    `json:"leverancier_id"`
	Naam          string `json:"naam"`
}

func (c *Client) GetAllProducten() ([]Product, error) {
	out := []Product{}
	err := c.Get("/api2/producten", &out)
	return out, err
}
