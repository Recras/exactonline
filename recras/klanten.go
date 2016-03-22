package recras

type Klant struct {
	ID          int    `json:"id"`
	Displaynaam string `json:"displaynaam"`
	Adres       string `json:"adres"`
	Postcode    string `json:"postcode"`
	Plaats      string `json:"plaats"`
}
