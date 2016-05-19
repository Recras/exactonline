package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/Recras/exactonline/dal"
	"github.com/Recras/exactonline/libhttp"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

type dashboardData struct {
	CurrentUser     *dal.UserRow
	Hostname        string
	KoppelingActief bool
}

func (dd *dashboardData) setDashboardData(r *http.Request) bool {
	cookieStore := context.Get(r, "cookieStore").(*sessions.CookieStore)

	session, _ := cookieStore.Get(r, "exactonline-session")
	if cu, ok := session.Values["user"].(*dal.UserRow); !ok {
		return false
	} else {
		dd.CurrentUser = cu
	}

	parts := strings.Split(dd.CurrentUser.Email, "@")
	if len(parts) < 2 {
		return false
	}
	dd.Hostname = parts[1]

	db := context.Get(r, "db").(*sqlx.DB)
	if cred, err := dal.FindCredentialByRecrasHostname(db, dd.Hostname); err != nil {
		return false
	} else {
		dd.KoppelingActief = cred.ExactRefreshToken != nil && len(*cred.ExactRefreshToken) > 0
	}

	return true
}

func GetHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	data := struct {
		dashboardData
	}{}
	if !data.setDashboardData(r) {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/home.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tmpl.Execute(w, data)
}
