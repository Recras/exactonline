package synctool

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/url"
	"time"
)

import (
	"github.com/Recras/exactonline"
	"github.com/Recras/exactonline/httperror"
	"github.com/Recras/exactonline/odata2json"
	"github.com/Recras/exactonline/recras"
)

import (
	"github.com/Sirupsen/logrus"
)

func Sync(logentry *logrus.Entry, errc chan<- error, rcl *recras.Client, ecl *exactonline.Client, syncdate string) {
	logrus.Debug("get Recras bedrijven")
	bedrijven, err := rcl.GetBedrijven(nil)
	if err != nil {
		errc <- err
	}
	for _, b := range bedrijven {
		logentry := logrus.WithField("recras_bedrijf", b)
		err := syncBedrijf(logentry, errc, rcl, ecl, syncdate, b)
		if err != nil {
			errc <- err
		}
	}
}

func syncBedrijf(logentry *logrus.Entry, errc chan<- error, rcl *recras.Client, ecl *exactonline.Client, syncdate string, b recras.Bedrijf) error {
	err := ecl.SetDivisionByVATNumber(b.BTWNummer)
	if err == exactonline.ErrDivisionNotFound {
		logentry.Debug("Skipping: no matching BTWNummer")
		errc <- errors.New(fmt.Sprintf("Bedrijf %s wordt overgeslagen: geen matchend BTW-nummer `%s` in Exact Online", b.Bedrijfsnaam, b.BTWNummer))
		return nil
	}
	if err != nil {
		logentry.WithField(
			"btw_nummer",
			b.BTWNummer,
		).Debugf("Error finding Division, %#v", err)
		return err
	}

	pc, err := ecl.FindPaymentConditionByDescription("recras")
	if err != nil {
		logentry.Warnf("Error finding PaymentCondition `recras`")
		return err
	}

	vcs, err := ecl.GetRecrasVATCodes()
	if err != nil {
		logentry.Warnf("Error retrieving VATCodes: %#v", err)
		return err
	}
	vatcodes := make(exactonline.VATCodeList)
	for _, vc := range vcs {
		var i float64
		fmt.Sscanf(vc.Description, "recras:%f", &i)
		vatcodes[i] = vc.Code
	}

	if err := syncProducten(logentry, errc, rcl, ecl); err != nil {
		logentry.Warnf("Error syncing producten")
		return err
	}

	ffilt := url.Values{}
	ffilt.Set("status", "verzonden,deels_betaald,betaald")
	ffilt.Set("datumNa", syncdate)
	ffilt.Set("embed", "regels,Klant")
	ffilt.Set("bedrijf_id", fmt.Sprintf("%d", b.ID))
	facturen, err := rcl.GetFacturenFilter(ffilt)
	if err != nil {
		logentry.Debugf("Error fetching facturen: %s", err)
		return err
	}
	logentry.WithField("#facturen", len(facturen)).WithField("startdatum", syncdate).Debug("Aantal facturen")
	for _, f := range facturen {
		err := syncFactuur(logentry.WithField("factuur", f.FactuurNummer), errc, rcl, ecl, f, pc, vatcodes)
		if err != nil {
			errc <- errors.New("Fout bij het kopieren van factuur " + f.FactuurNummer + ": " + err.Error())
		}
	}

	logentry.Info("Finished bedrijf sync")
	return nil
}

func syncProducten(logentry *logrus.Entry, errc chan<- error, rcl *recras.Client, ecl *exactonline.Client) error {
	producten, err := rcl.GetAllProducten()
	if err != nil {
		logentry.Warnf("Error retrieving producten from Recras")
		return err
	}
	for _, p := range producten {
		_, err := syncProduct(logentry.WithField("RecrasProduct", p.ID), p, ecl)
		if err != nil {
			logentry.WithFields(logrus.Fields{
				"RecrasProduct": p.ID,
				"error":         err,
			}).Warn("Error syncing Product")
			errc <- err
		}
	}
	return nil
}

func syncProduct(logentry *logrus.Entry, p recras.Product, ecl *exactonline.Client) (exactonline.Item, error) {
	item, err := ecl.FindItemByRecrasID(p.ID)
	if err == nil {
		logentry.Info("Found item")
		return item, nil
	} else if _, ok := err.(exactonline.ErrItemNotFound); !ok {
		return exactonline.Item{}, err
	}
	item.Code = fmt.Sprintf("recras%d", p.ID)
	item.Description = fmt.Sprintf("Recras p%d: %s", p.ID, p.Naam)
	item.StartDate = odata2json.Date{time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)}
	item.IsSalesItem = true
	item.Unit = "recras"
	if err := item.Save(ecl); err != nil {
		if e, ok := err.(httperror.HTTPError); ok {
			bb := bytes.NewBuffer(nil)
			io.Copy(bb, e.Response.Body)
			bb2 := bytes.NewBuffer(nil)
			io.Copy(bb2, e.Request.Body)
			logentry.Debug("response" + bb.String())
			logentry.Debug("request" + bb2.String())
		}
		return exactonline.Item{}, err
	}
	return item, nil
}

func syncFactuur(logentry *logrus.Entry, errc chan<- error, rcl *recras.Client, ecl *exactonline.Client, f recras.Factuur, pc exactonline.PaymentCondition, vatcodes exactonline.VATCodeList) error {
	_, err := ecl.FindSalesEntry(f.FactuurNummer)
	if err == nil { // SalesEntry found
		logentry.Info("SalesEntry exists")
		return nil
	} else if _, ok := err.(*exactonline.ErrSalesEntryNotFound); !ok {
		logentry.Warnf("Skipping: Error retrieving salesentry: %#v", err)
		return err
	}

	lines, err := convertFactuurregels(ecl, f.Regels, 1, vatcodes)
	if err != nil {
		logentry.Warnf("Skipping: Error converting factuurregels: %#v", err)
		return err
	}
	if len(lines) == 0 {
		logentry.Info("Skipping: No lines with value")
		errc <- errors.New(fmt.Sprintf("Skipping invoice %s, no lines with value", f.FactuurNummer))
		return nil
	}

	cust, err := ecl.FindAccountByRecrasID(f.Klant.ID)
	if _, ok := err.(exactonline.ErrAccountNotFound); ok {
		_, e := syncKlant(logentry.WithField("klant", f.KlantID), errc, ecl, f.Klant)
		if e != nil {
			logentry.WithField("klant", f.KlantID).Warnf("Error saving Klant: %#v", err)
			return e
		}
		cust, _ = ecl.FindAccountByRecrasID(f.Klant.ID)
	} else if err != nil {
		logentry.WithField("klant", f.KlantID).Warnf("Error finding Klant: %#v", err)
		return err
	}
	logentry.WithField("account", cust).Debug("Account")

	entry := convertFactuur(ecl, cust, f, pc)
	entry.SetSalesEntryLines(lines)

	pdf, err := uploadFactuurPDF(rcl, ecl, f, cust)
	if err != nil {
		logentry.Warnf("Error uploading factuur PDF: %#v", err)
	} else {
		_ = pdf
		entry.Document = pdf.ID
	}

	if err := entry.Save(ecl); err != nil {
		if e, ok := err.(httperror.HTTPError); ok {
			reqbody, _ := ioutil.ReadAll(e.Request.Body)
			resbody, _ := ioutil.ReadAll(e.Response.Body)
			logentry.WithFields(logrus.Fields{
				"request body":  string(reqbody),
				"response body": string(resbody),
				"salesentry":    entry,
			}).Warnf("HTTP error saving factuur: %#v", e)
		}
		logentry.WithField("salesentry", entry).Warnf("Error saving factuur: %#v", err)
		return err
	}
	logentry.Info("Saved factuur")
	return nil
}

func syncKlant(logentry *logrus.Entry, errc chan<- error, ecl *exactonline.Client, k recras.Klant) (exactonline.Account, error) {

	a, err := ecl.FindAccountByRecrasID(k.ID)
	if _, ok := err.(exactonline.ErrAccountNotFound); ok {
		logentry.Info("Creating account")
		a = exactonline.Account{
			Name:         fmt.Sprintf("K%d %s", k.ID, k.Displaynaam),
			AddressLine1: k.Adres,
			Postcode:     k.Postcode,
			City:         k.Plaats,
			SearchCode:   fmt.Sprintf("K%d", k.ID),
			Status:       "C",
		}
		if a.Name == "" {
			a.Name = fmt.Sprintf("Recras K%d", k.ID)
		}
		if err := a.Save(ecl); err != nil {
			return exactonline.Account{}, err
		}
	}
	return a, nil
}

type ErrNoGLRevenueAccount struct {
	ProductID int
}

func (err ErrNoGLRevenueAccount) Error() string {
	return fmt.Sprintf("Item with ProductID %d has no GLRevenue account", err.ProductID)
}

type ErrNoVATCode struct {
	Percentage float64
}

func (err ErrNoVATCode) Error() string {
	return fmt.Sprintf("No VATCode specified for percentage %f", err.Percentage)
}

func convertFactuurregels(exact_itemfinder exactonline.ItemFinder, r []recras.Factuurregel, reductionfactor float64, vatcodes exactonline.VATCodeList) ([]exactonline.SalesEntryLine, error) {
	out := []exactonline.SalesEntryLine{}
	for _, regel := range r {
		if regel.Type == recras.FactuurregelItem {
			i, err := exact_itemfinder.FindItemByRecrasID(regel.ProductID)
			if err != nil {
				return nil, err
			}
			amountFC := float64(regel.Aantal) * regel.Bedrag * (100 - regel.Kortingspercentage) / 100 * reductionfactor
			if math.Abs(amountFC) < 1e-3 {
				return out, nil
			}
			if i.GLRevenue == "" {
				return nil, ErrNoGLRevenueAccount{
					ProductID: regel.ProductID,
				}
			}
			vc, ok := vatcodes[regel.BTWPercentage]
			if !ok {
				return nil, ErrNoVATCode{Percentage: regel.BTWPercentage}
			}
			line := exactonline.SalesEntryLine{
				AmountFC:    amountFC,
				GLAccount:   i.GLRevenue,
				Description: regel.Naam,
				Quantity:    float64(regel.Aantal),
				VATCode:     vc,
			}
			out = append(out, line)
		} else if regel.Type == recras.FactuurregelGroep {
			lines, err := convertFactuurregels(exact_itemfinder, regel.Regels, reductionfactor*(100-regel.Kortingspercentage)/100, vatcodes)
			if err != nil {
				return nil, err
			}
			out = append(out, lines...)
		}
	}
	return out, nil
}

func convertFactuur(exact_itemfinder exactonline.ItemFinder, a exactonline.Account, f recras.Factuur, pc exactonline.PaymentCondition) exactonline.SalesEntry {
	entry := exactonline.SalesEntry{
		Description:      "Recras factuur: " + f.FactuurNummer,
		PaymentReference: f.FactuurNummer,
		Journal:          "recras",
		Customer:         a.ID,
		YourRef:          f.ReferentieKlant,
		PaymentCondition: pc.Code,
	}
	if (f.Datum.Time != time.Time{}) {
		entry.EntryDate.Time = f.Datum.Time
		entry.DueDate.Time = entry.EntryDate.Time.AddDate(0, 0, f.Betaaltermijn)
	}
	if f.CalculatedTotaalbedragInclusiefBTW < 0 {
		entry.Type = exactonline.SalesEntryCreditNote
	}
	return entry
}

func uploadFactuurPDF(r *recras.Client, ecl *exactonline.Client, f recras.Factuur, ac exactonline.Account) (*exactonline.Document, error) {
	ret := new(exactonline.Document)
	ret.Subject = "Recras factuur " + f.FactuurNummer

	dt, err := ecl.FindDocumentTypeByDescription("Sales invoice")
	if err != nil {
		return nil, err
	}
	ret.Type = dt.ID

	ret.Account = ac.ID
	if err := ret.Save(ecl); err != nil {
		return nil, err
	}

	resp, err := r.Client.Get("/facturen/" + f.PdfLocatie)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var bb bytes.Buffer
	_, err = bb.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	da := exactonline.DocumentAttachment{
		Document:   ret.ID,
		FileName:   f.PdfLocatie,
		Attachment: bb.Bytes(),
	}
	da.Save(ecl)

	return ret, nil
}
