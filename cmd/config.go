package cmd

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/matrix-org/gomatrix"
	gorp "gopkg.in/gorp.v2"

	"github.com/securityfirst/matrix-notifier/database"
	"github.com/securityfirst/matrix-notifier/server"
)

type config struct {
	DB struct {
		Username string
		Password string
		Host     string
		Database string
		Options  string
		Debug    bool
	}
	Matrix struct {
		Address string
	}
	Mailer struct {
		Address  string
		Username string
		Password string
		From     string
		Subject  string
		Body     string
	}
	Server struct {
		Address string
		Secret  string
		Debug   bool
	}
}

func (c config) Init() {
	if !c.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
}

func (c config) GetMailer() (server.Mailer, error) {
	from, err := mail.ParseAddress(c.Mailer.From)
	if err != nil {
		return nil, err
	}
	subject, err := template.New("subject").Parse(c.Mailer.Subject)
	if err != nil {
		return nil, err
	}
	body, err := template.New("body").Parse(c.Mailer.Body)
	if err != nil {
		return nil, err
	}
	return &server.SMTPMailer{
		Address:  c.Mailer.Address,
		Username: c.Mailer.Username,
		Password: c.Mailer.Password,
		From:     from,
		Subject:  subject,
		Body:     body,
	}, nil
}

func (c config) GetDB() (*gorp.DbMap, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?%s",
		c.DB.Username, c.DB.Password, c.DB.Host, c.DB.Database, c.DB.Options))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	dbMap := gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	if c.DB.Debug {
		dbMap.TraceOn("[gorp] ", log.New(os.Stdout, "", 0))
	}
	database.InitDBMap(&dbMap)

	return &dbMap, nil
}

func (c config) GetMatrix() (*gomatrix.Client, error) {
	return gomatrix.NewClient(c.Matrix.Address, "", "")
}
