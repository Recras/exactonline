package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"time"
)

import (
	"github.com/Recras/exactonline/dal"
	"github.com/Recras/exactonline/handlers"
	"github.com/Recras/exactonline/libenv"
	"github.com/Recras/exactonline/middlewares"
)

import (
	"github.com/Sirupsen/logrus"
	"github.com/carbocation/interpose"
	gorilla_mux "github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattes/migrate/driver/postgres"
	"github.com/mattes/migrate/migrate"
	"github.com/tylerb/graceful"
)

func init() {
	gob.Register(&dal.UserRow{})
	logrus.SetLevel(logrus.DebugLevel)
}

// NewApplication is the constructor for Application struct.
func NewApplication() (*Application, error) {
	db_user := libenv.EnvWithDefault("POSTGRES_ENV_POSTGRES_USER", "exactonline")
	db_pass := libenv.EnvWithDefault("POSTGRES_ENV_POSTGRES_PASSWORD", "")
	db_host := libenv.EnvWithDefault("POSTGRES_PORT_5432_TCP_ADDR", "localhost")
	db_port := libenv.EnvWithDefault("POSTGRES_PORT_5432_TCP_PORT", "5432")

	if db_user == "" || db_pass == "" {
		return nil, errors.New("Be sure to set POSTGRES_ENV_POSTGRES_USER and POSTGRES_ENV_POSTGRES_PASSWORD environment variables")
	}
	exact_client_secret := libenv.EnvWithDefault("EXACT_CLIENT_SECRET", "")
	if exact_client_secret == "" {
		return nil, errors.New("Be sure to set EXACT_CLIENT_SECRET environment variable")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/exactonline?sslmode=disable", db_user, db_pass, db_host, db_port)

	fmt.Println("Running migrations...")
	errors, ok := migrate.UpSync(dsn, "./migrations")
	if !ok {
		logrus.Fatal(errors)
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	cookieStoreSecret := libenv.EnvWithDefault("COOKIE_SECRET", "4iKivAZAZORgZ3ya")

	app := &Application{}
	app.dsn = dsn
	app.db = db
	app.cookieStore = sessions.NewCookieStore([]byte(cookieStoreSecret))

	return app, err
}

// Application is the application object that runs HTTP server.
type Application struct {
	dsn         string
	db          *sqlx.DB
	cookieStore *sessions.CookieStore
}

func (app *Application) middlewareStruct() (*interpose.Middleware, error) {
	middle := interpose.New()
	middle.Use(middlewares.SetDB(app.db))
	middle.Use(middlewares.SetCookieStore(app.cookieStore))

	middle.UseHandler(app.mux())

	return middle, nil
}

func (app *Application) mux() *gorilla_mux.Router {
	MustLogin := middlewares.MustLogin

	router := gorilla_mux.NewRouter()

	router.Handle("/", MustLogin(http.HandlerFunc(handlers.GetHome))).Methods("GET")

	// creating a new Exact Online Link
	router.Handle("/link_exact", MustLogin(http.HandlerFunc(handlers.GetLinkExact))).Methods("GET")
	router.Handle("/link_exact", MustLogin(http.HandlerFunc(handlers.PostLinkExact))).Methods("POST")
	router.Handle("/callback", MustLogin(http.HandlerFunc(handlers.GetCallback))).Methods("GET")

	// manage existing link
	router.Handle("/status", MustLogin(http.HandlerFunc(handlers.GetStatus))).Methods("GET")
	router.Handle("/sync", MustLogin(http.HandlerFunc(handlers.GetSync))).Methods("GET")

	router.HandleFunc("/login", handlers.GetLogin).Methods("GET")
	router.HandleFunc("/login", handlers.PostLogin).Methods("POST")
	router.HandleFunc("/logout", handlers.GetLogout).Methods("GET")

	// Path of static files must be last!
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	return router
}

func main() {
	app, err := NewApplication()
	if err != nil {
		logrus.Fatal(err.Error())
	}

	middle, err := app.middlewareStruct()
	if err != nil {
		logrus.Fatal(err.Error())
	}

	app.runTimedSync()

	serverAddress := libenv.EnvWithDefault("HTTP_ADDR", "127.0.0.1:8888")
	certFile := libenv.EnvWithDefault("HTTP_CERT_FILE", "")
	keyFile := libenv.EnvWithDefault("HTTP_KEY_FILE", "")
	drainIntervalString := libenv.EnvWithDefault("HTTP_DRAIN_INTERVAL", "1s")

	drainInterval, err := time.ParseDuration(drainIntervalString)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	srv := &graceful.Server{
		Timeout: drainInterval,
		Server:  &http.Server{Addr: serverAddress, Handler: middle},
	}

	logrus.Infoln("Running HTTP server on " + serverAddress)
	if certFile != "" && keyFile != "" {
		err = srv.ListenAndServeTLS(certFile, keyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		logrus.Fatal(err.Error())
	}
}

const beginHour = 3
const beginMinute = 7

func (app *Application) runTimedSync() {
	nexttick := time.Now()

	logrus.Debugf("Waiting for %s", nexttick.Sub(time.Now()))
	time.AfterFunc(nexttick.Sub(time.Now()), func() {
		logrus.Debugf("Running sync")
		ticker := time.Tick(24 * time.Hour)

		syncAll := func(entry *logrus.Entry) {
			creds, err := dal.FindAllCredentials(app.db)
			if err != nil {
				entry.WithField("error", err).Error("Error finding credentials")
				return
			}

			for _, cred := range creds {
				handlers.SyncRecras(&cred, entry.WithField("recras_hostname", cred.RecrasHostname))
			}
		}

		syncAll(logrus.WithField("timed_sync", time.Now()))

		for t := range ticker {
			entry := logrus.WithField("timed_sync", t)
			if t.After(time.Now().Add(time.Hour)) {
				entry.Warnf("It is already %s, skipping sync", time.Now())
				continue
			}
			entry.Info("starting scheduled sync")
			syncAll(entry)
		}
	})
}
