package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/Recras/exactonline"
	"github.com/Recras/exactonline/dal"
	"github.com/Recras/exactonline/recras"
	"github.com/Recras/exactonline/synctool"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

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
	//recras_username := parts[0]
	recras_hostname := parts[1]
	cred, err := dal.FindCredentialByRecrasHostname(db, recras_hostname)
	if err != nil {
		logrus.Infof("handlers.GetStatus: error retrieving credentials for `%s`: %s", recras_hostname, err)
	}

	cl := exactonline.EnvConfig().NewClient(oauth2.Token{RefreshToken: *cred.ExactRefreshToken})
	err = cl.GetDefaultDivision()
	if err != nil {
		logrus.Errorf("handlers.GetStatus: error retrieving currentdivision: %s", err)
	}
	fmt.Fprintf(w, "CurrentDivision: %d\n", cl.Division)

	producten, err := cl.GetAllItems()
	if err != nil {
		logrus.Errorf("handlers.GetStatus: error retrieving Items: %s", err)
		return
	}
	fmt.Fprintf(w, "Producten: %s\n", producten)
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
