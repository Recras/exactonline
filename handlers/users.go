package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/Recras/exactonline/dal"
	"github.com/Recras/exactonline/libhttp"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

import (
	"github.com/Recras/exactonline/recras"
)

func GetLoginWithoutSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	tmpl, err := template.ParseFiles("templates/users/login-signup-parent.html.tmpl", "templates/users/login.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tmpl.Execute(w, nil)
}

// GetLogin get login page.
func GetLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	cookieStore := context.Get(r, "cookieStore").(*sessions.CookieStore)

	session, _ := cookieStore.Get(r, "exactonline-session")

	currentUserInterface := session.Values["user"]
	if currentUserInterface != nil {
		fmt.Printf("%#v", currentUserInterface)
		http.Redirect(w, r, "/", 302)
		return
	}

	GetLoginWithoutSession(w, r)
}

// PostLogin performs login.
func PostLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	cookieStore := context.Get(r, "cookieStore").(*sessions.CookieStore)

	email := r.FormValue("Email")
	password := r.FormValue("Password")

	userInfo := strings.Split(email, "@")
	user, recrasHostname := userInfo[0], userInfo[1]
	if err := recras.IsValidUser(recrasHostname, user, password); err == recras.ErrInvalidCredentials {
		libhttp.HandleErrorJson(w, err)
		return
	} else if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	session, _ := cookieStore.Get(r, "exactonline-session")
	session.Values["user"] = dal.UserRow{
		Email: email,
	}

	db := context.Get(r, "db").(*sqlx.DB)
	_, err := dal.CreateCredential(db, recrasHostname, user, password)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
	}

	err = session.Save(r, w)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	http.Redirect(w, r, "/", 302)
}

func GetLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	cookieStore := context.Get(r, "cookieStore").(*sessions.CookieStore)

	session, _ := cookieStore.Get(r, "exactonline-session")

	delete(session.Values, "user")
	session.Save(r, w)

	http.Redirect(w, r, "/login", 302)
}
