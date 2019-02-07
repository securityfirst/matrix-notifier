package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	gorp "gopkg.in/gorp.v2"

	"github.com/securityfirst/matrix-notifier/database"
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
	Server struct {
		Address string
		Debug   bool
	}
}

func (c config) Init() {
	if !c.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
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
		dbMap.TraceOn("[DB] ", log.New(os.Stdout, "", 0))
	}
	if err := database.InitDBMap(&dbMap); err != nil {
		return nil, err
	}
	return &dbMap, nil
}
