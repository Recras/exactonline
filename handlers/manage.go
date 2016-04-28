package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"text/template"

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
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
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

	data := struct {
		CurrentUser     *dal.UserRow
		Administrations []administrationData
		GeneralError    string
	}{
		CurrentUser:     u,
		Administrations: []administrationData{},
	}

	cred, err := dal.FindCredentialByRecrasHostname(db, recras_hostname)
	if err != nil {
		logger.Infof("error retrieving credentials: %s", err)
	}
	if cred.ExactRefreshToken == nil {
		logger.Error("no ExactRefreshToken")
		data.GeneralError = "Verbinding met Exact Online is nog niet gelegd"
		w.WriteHeader(400)
		tmpl.Execute(w, data)
		return
	}

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

		checkDagboek(cl, adminStatus, bedrijflogger)

		adminStatus.EverythingOK = adminStatus.DefaultItemGroupOK && adminStatus.DefaultJournalOK
	}

	tmpl.Execute(w, data)
}

func checkDagboek(cl *exactonline.Client, ad *administrationData, logger *logrus.Entry) {
	j, err := cl.FindDefaultJournal()
	if err != nil && err != exactonline.ErrJournalNotFound {
		logrus.Errorf("%#v", err)
	}
	ad.DefaultJournalOK = (err == nil)
	ad.DefaultJournalDesc = j.Description
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
		fmt.Fprintln(messagebuf, e.Error())
	}
	//fmt.Fprint(w, messagebuf.String())

	personeel, _ := rcl.GetCurrentPersoneel()

	//fmt.Fprintf(w, "Personeel: %#v", personeel)

	contactmoment := recras.Contactmoment{
		ContactID:        personeel.ID,
		ContactpersoonID: personeel.ContactpersoonID,
		Onderwerp:        "Synchronisatierapport",
		Bericht:          messagebuf.String(),
		Soort:            "noot",
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
	//recras_username := parts[0]
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
