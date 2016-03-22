package handlers

import (
	"net/http"
	"text/template"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"

	"github.com/Recras/exactonline"
	"github.com/Recras/exactonline/dal"
	"github.com/Recras/exactonline/libhttp"
)

func callbackError(w http.ResponseWriter, r *http.Request, e error) {
	cookieStore := context.Get(r, "cookieStore").(*sessions.CookieStore)

	session, _ := cookieStore.Get(r, "exactonline-session")
	currentUser, _ := session.Values["user"].(*dal.UserRow)
	data := struct {
		CurrentUser *dal.UserRow
		Error       error
	}{
		currentUser,
		e,
	}
	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/callback_error.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}
	tmpl.Execute(w, data)
}
func GetCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	cookieStore := context.Get(r, "cookieStore").(*sessions.CookieStore)

	session, _ := cookieStore.Get(r, "exactonline-session")
	_, ok := session.Values["user"].(*dal.UserRow)
	if !ok {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	state := r.URL.Query().Get("state")

	db := context.Get(r, `db`).(*sqlx.DB)
	cred, err := dal.FindCredentialByState(db, state)
	if err != nil {
		callbackError(w, r, err)
		return
	}

	code := r.URL.Query().Get("code")
	tok, err := exactonline.EnvConfig().Exchange(code)
	if err != nil {
		callbackError(w, r, err)
		return
	}
	err = cred.UpdateToken(db, tok.AccessToken, tok.RefreshToken)
	if err != nil {
		callbackError(w, r, err)
		return
	}

	http.Redirect(w, r, "/status", http.StatusFound)
}
