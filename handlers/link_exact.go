package handlers

import (
	"net/http"
	"strings"
	"text/template"
)
import (
	"github.com/Recras/exactonline"
	"github.com/Recras/exactonline/dal"
	"github.com/Recras/exactonline/libhttp"
)
import (
	"github.com/gorilla/context"
	"github.com/jmoiron/sqlx"
)

func GetLinkExact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	data := struct {
		dashboardData
	}{}
	if !data.setDashboardData(r) {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/link_exact.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tmpl.Execute(w, data)
}

func PostLinkExact(w http.ResponseWriter, r *http.Request) {
	data := struct {
		dashboardData
		SystemError error
	}{}
	if !data.setDashboardData(r) {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	parts := strings.Split(data.CurrentUser.Email, "@")
	recras_username := parts[0]
	recras_hostname := parts[1]
	recras_password := data.CurrentUser.Password

	db := context.Get(r, "db").(*sqlx.DB)
	cred, err := dal.CreateCredential(db, recras_hostname, recras_username, recras_password)
	if err != nil {
		data.SystemError = err
		tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/error.html.tmpl")
		if err != nil {
			libhttp.HandleErrorJson(w, err)
			return
		}

		tmpl.Execute(w, data)
		return
	}

	http.Redirect(w, r, exactonline.EnvConfig().Oauth.AuthCodeURL(cred.State), http.StatusFound)
}
