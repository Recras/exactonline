package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/Recras/exactonline"
	"github.com/Recras/exactonline/dal"
	"github.com/Recras/exactonline/libhttp"
	"github.com/Recras/exactonline/recras"
	"github.com/Recras/exactonline/synctool"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

type administrationData struct {
	RecrasBedrijfNaam string
	Error             string
	EverythingOK      bool

	DefaultItemGroupOK   bool
	DefaultItemGroupCode string

	DefaultJournalOK   bool
	DefaultJournalDesc string

	VATCodesOK bool
	VATCodes   map[string]bool

	PaymentConditionOK bool
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	data := struct {
		dashboardData
		Administrations []administrationData
		GeneralError    string
	}{}
	data.Administrations = []administrationData{}

	if !data.setDashboardData(r) {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	recras_hostname := data.Hostname

	logger := logrus.WithFields(logrus.Fields{
		"function":        "handlers.GetStatus",
		"recras_hostname": recras_hostname,
	})

	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/status.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		logger.Errorf("template parsing error %s", err)
		return
	}

	if !data.KoppelingActief {
		data.GeneralError = "Verbinding met Exact Online is nog niet gelegd"
		w.WriteHeader(400)
		tmpl.Execute(w, data)
		return
	}

	db := context.Get(r, "db").(*sqlx.DB)
	cred, err := dal.FindCredentialByRecrasHostname(db, recras_hostname)
	tok := oauth2.Token{
		RefreshToken: *cred.ExactRefreshToken,
	}
	cl := exactonline.EnvConfig().NewClient(tok)
	err = cl.GetDefaultDivision()
	if err != nil {
		logger.Errorf("error retrieving currentdivision: %s", err)
		w.WriteHeader(500)
		data.GeneralError = "Verbindingsfout met Exact Online"
		tmpl.Execute(w, data)
		return
	}

	rcl := recras.NewClient(cred.RecrasHostname, cred.RecrasUsername, cred.RecrasPassword)
	bedrijven, err := rcl.GetBedrijven(nil)
	if err != nil {
		logger.Errorf("error retrieving bedrijven: %s", err)
		w.WriteHeader(500)
		data.GeneralError = "Verbindingsfout met Recras"
		tmpl.Execute(w, data)
		return
	}

	var btw_percentages struct {
		Waarde string `json:"waarde"`
	}
	rcl.Get("/api2/instellingen/btw_percentages", &btw_percentages)
	percentages := strings.Split(btw_percentages.Waarde, ",")

	for _, b := range bedrijven {
		data.Administrations = append(data.Administrations, administrationData{
			RecrasBedrijfNaam: b.Bedrijfsnaam,
		})
		adminStatus := &data.Administrations[len(data.Administrations)-1]

		err = cl.SetDivisionByVATNumber(b.BTWNummer)
		if err == exactonline.ErrDivisionNotFound {
			adminStatus.Error = "Geen administratie in Exact Online met BTW-nummer " + b.BTWNummer
			continue
		}
		bedrijflogger := logger.WithField("exact_administration_id", cl.Division)
		bedrijflogger.Info("Bedrijf gevonden")

		itemgroup, err := cl.FindDefaultItemGroup()
		if err != nil {
			bedrijflogger.Info(err.Error())
		}
		adminStatus.DefaultItemGroupOK = (err == nil)
		adminStatus.DefaultItemGroupCode = itemgroup.Code

		checkDefaultJournal(cl, adminStatus, bedrijflogger)
		checkVATCodes(percentages, cl, adminStatus, bedrijflogger)
		checkPaymentCondition(cl, adminStatus, bedrijflogger)

		adminStatus.EverythingOK = adminStatus.DefaultItemGroupOK && adminStatus.DefaultJournalOK && adminStatus.VATCodesOK && adminStatus.PaymentConditionOK
	}

	tmpl.Execute(w, data)
}

func checkDefaultJournal(cl *exactonline.Client, ad *administrationData, logger *logrus.Entry) {
	j, err := cl.FindDefaultJournal()
	if err != nil && err != exactonline.ErrJournalNotFound {
		logrus.Errorf("%#v", err)
	}
	ad.DefaultJournalOK = (err == nil)
	ad.DefaultJournalDesc = j.Description
}

func checkVATCodes(percentages []string, cl *exactonline.Client, ad *administrationData, logger *logrus.Entry) {
	codes, err := cl.GetRecrasVATCodes()
	if err != nil {
		logrus.Errorf("VATCodes: %#v", err)
		return
	}
	ad.VATCodes = make(map[string]bool)
	ad.VATCodesOK = true
	for _, p := range percentages {
		code := "recras:" + p
		ad.VATCodes[code] = false
		for cidx := range codes {
			if codes[cidx].Description == code {
				ad.VATCodes[code] = true
			}
		}
		if !ad.VATCodes[code] {
			ad.VATCodesOK = false
		}
	}
}

func checkPaymentCondition(cl *exactonline.Client, ad *administrationData, logger *logrus.Entry) {
	_, err := cl.FindPaymentConditionByDescription("recras")
	if err != nil {
		logrus.Errorf("PaymentCondition: %#v", err)
	}
	ad.PaymentConditionOK = (err == nil)
}

func SyncRecras(cred *dal.Credential, entry *logrus.Entry) {
	cl := exactonline.EnvConfig().NewClient(oauth2.Token{RefreshToken: *cred.ExactRefreshToken})
	err := cl.GetDefaultDivision()
	if err != nil {
		entry.Errorf("handlers.GetStatus: error retrieving currentdivision: %s", err)
	}

	rcl := recras.NewClient(cred.RecrasHostname, cred.RecrasUsername, cred.RecrasPassword)

	errc := make(chan error)
	go func() {
		f := cred.StartSync.Format("2006-01-02")
		synctool.Sync(entry, errc, &rcl, cl, f)
		close(errc)
	}()
	messagebuf := bytes.NewBuffer(nil)
	for e := range errc {
		logrus.Infof(e.Error())
		fmt.Fprintf(messagebuf, "%s<br>\n", e.Error())
	}

	personeel, _ := rcl.GetCurrentPersoneel()
	gebruiker, _ := rcl.GetGebruiker(personeel)
	now := time.Now()

	contactmoment := recras.Contactmoment{
		ContactID:               personeel.ID,
		ContactpersoonID:        personeel.ContactpersoonID,
		Onderwerp:               "Synchronisatierapport",
		Bericht:                 messagebuf.String(),
		Soort:                   "noot",
		ContactOpnemen:          &now,
		ContactOpnemenGroup:     gebruiker.GetFirstRolId(),
		ContactOpnemenOpmerking: "Synchronisatierapport Exact Online",
	}
	err = contactmoment.Save(&rcl)
	if err != nil {
		entry.WithField("recras_personeel", personeel.Displaynaam).Error("error saving contactmoment: " + err.Error())
	}
	entry.Info("Synchronization done")
}

func GetSync(w http.ResponseWriter, r *http.Request) {
	cookieStore := context.Get(r, "cookieStore").(*sessions.CookieStore)

	session, _ := cookieStore.Get(r, "exactonline-session")
	u, ok := session.Values["user"].(*dal.UserRow)
	if !ok {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	db := context.Get(r, "db").(*sqlx.DB)

	parts := strings.Split(u.Email, "@")
	recras_hostname := parts[1]
	entry := logrus.WithFields(logrus.Fields{
		"recras_hostname": recras_hostname,
	})
	cred, err := dal.FindCredentialByRecrasHostname(db, recras_hostname)
	if err != nil {
		entry.Errorf("handlers.GetStatus: error retrieving credentials: %s", err)
		fmt.Fprintf(w, "Inloggegevens Exact Online konden niet gevonden worden voor %s", recras_hostname)
	}

	SyncRecras(cred, entry)
}
