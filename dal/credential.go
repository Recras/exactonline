package dal

import (
	"database/sql"
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
)

type Credential struct {
	RecrasHostname string `db:"recras_hostname"`
	RecrasUsername string `db:"recras_username"`
	RecrasPassword string `db:"recras_password"`

	ExactAccessToken  *string `db:"exact_access_token"`
	ExactRefreshToken *string `db:"exact_refresh_token"`

	State     string    `db:"state"`
	StartSync time.Time `db:"start_sync_date"`
}

func FindAllCredentials(db *sqlx.DB) ([]Credential, error) {
	cred := []Credential{}
	err := db.Select(&cred, `SELECT * FROM credential`)
	return cred, err
}

func FindCredentialByState(db *sqlx.DB, state string) (*Credential, error) {
	cred := &Credential{}
	err := db.Get(cred, `SELECT * FROM credential WHERE state=$1`, state)
	return cred, err
}

func FindCredentialByRecrasHostname(db *sqlx.DB, hostname string) (*Credential, error) {
	cred := &Credential{}
	err := db.Get(cred, `SELECT * FROM credential WHERE recras_hostname=$1`, hostname)
	return cred, err
}

type CredentialError struct {
	part string
	orig error
}

func (err CredentialError) Error() string {
	return "createCredential " + err.part + ": " + err.orig.Error()
}

func CreateCredential(db *sqlx.DB, recrasHostname, recrasUsername, recrasPassword string) (*Credential, error) {
	cred := &Credential{}
	err := db.Get(cred, `SELECT * FROM credential WHERE recras_hostname=$1`, recrasHostname)
	if err == sql.ErrNoRows {
	} else if err != nil {
		return nil, CredentialError{"select", err}
	} else {
		return cred, nil
	}

	state := make([]byte, 10)
	chars := `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`
	for i := 0; i < 10; i++ {
		state[i] = chars[rand.Intn(len(chars))]
	}

	cred.State = string(state)
	cred.RecrasHostname = recrasHostname
	cred.RecrasUsername = recrasUsername
	cred.RecrasPassword = recrasPassword
	cred.StartSync = time.Now()

	stmt, err := db.PrepareNamed(`INSERT INTO credential (recras_hostname, recras_username, recras_password, state, start_sync_date) VALUES(:recras_hostname, :recras_username, :recras_password, :state, :start_sync_date)`)
	if err != nil {
		return nil, CredentialError{"prepareInsert", err}
	}
	_, err = stmt.Exec(cred)
	if err != nil {
		return nil, CredentialError{"insert", err}
	}
	return cred, nil
}

func (c *Credential) UpdateToken(db *sqlx.DB, accessToken, refreshToken string) error {
	c.ExactAccessToken = &accessToken
	c.ExactRefreshToken = &refreshToken

	stmt, err := db.PrepareNamed(`UPDATE credential SET exact_access_token=:exact_access_token, exact_refresh_token=:exact_refresh_token WHERE recras_hostname=:recras_hostname`)
	if err != nil {
		return CredentialError{"prepareUpdateToken", err}
	}
	_, err = stmt.Exec(c)
	if err != nil {
		return CredentialError{"updateToken", err}
	}
	return nil
}
