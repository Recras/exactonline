package recras

import (
	"net/url"
	"time"
)

type Date struct {
	Time time.Time
}

func (d *Date) UnmarshalJSON(b []byte) error {
	t, err := time.Parse(`"2006-01-02"`, string(b))
	d.Time = t
	return err
}

const (
	FactuurregelGroep = "groep"
	FactuurregelItem  = "item"
)

const (
	StatusVerzonden    = "verzonden"
	StatusConcept      = "concept"
	StatusDeelsBetaald = "deels_betaald"
	StatusBetaald      = "betaald"
)

type Factuurregel struct {
	ID                   int            `json:"id"`
	Naam                 string         `json:"naam"`
	Type                 string         `json:"type"`
	Kortingspercentage   float64        `json:"kortingspercentage"`
	Kortingsomschrijving string         `json":kortingomschrijving"`
	Aantal               int            `json:"aantal"`
	Bedrag               float64        `json:"bedrag"`
	BTWPercentage        float64        `json:"btw_percentage"`
	ProductID            int            `json:"product_id"`
	BoekingsregelID      int            `json:"boekingsregel_id"`
	Regels               []Factuurregel `json:"regels"`
}

type Factuur struct {
	ID                                 int            `json:"id"`
	KlantID                            int            `json:"klant_id"`
	Status                             string         `json:"status"`
	FactuurNummer                      string         `json:"factuur_nummer"`
	Datum                              Date           `json:"datum"`
	Betaaltermijn                      int            `json:"betaaltermijn"`
	BedrijfID                          int            `json:"bedrijf_id"`
	ReferentieKlant                    string         `json:"referentie_klant"`
	Regels                             []Factuurregel `json:"regels,omitempty"`
	Klant                              Klant          `json:"Klant,omitempty"`
	PdfLocatie                         string         `json:"pdf_locatie,omitempty"`
	CalculatedTotaalbedragInclusiefBTW float64        `json:"calculated_totaalbedrag_inclusief_btw"`
}

func (c *Client) GetFacturenFilter(f url.Values) ([]Factuur, error) {
	if f == nil {
		f = url.Values{}
	}
	out := []Factuur{}
	url := "/api2/facturen?regelsformat=exactonline&" + f.Encode()
	err := c.Get(url, &out)
	return out, err
}
