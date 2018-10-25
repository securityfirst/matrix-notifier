package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"reflect"

	"gopkg.in/gorp.v2"

	"github.com/gin-gonic/gin"
	"github.com/matrix-org/gomatrix"
)

// NewServer returns a new Server.
func NewServer(address string, db *gorp.DbMap, c *gomatrix.Client, m Mailer, secret []byte) *Server {
	engine := gin.Default()
	s := Server{
		server:   &http.Server{Addr: address, Handler: engine},
		db:       db,
		matrix:   c,
		mailer:   m,
		verifier: verifier(secret),
	}

	engine.GET("/_matrix/client/r0/organisation/:orgID/verify", s.RedirectToIntent())
	engine.POST("/_matrix/client/r0/organisation/:orgID/verify", s.ParseRequest(createUserRequest{}), s.CreateUser())

	auth := engine.Use(s.Authenticate())
	auth.POST("/_matrix/client/r0/organisation/:orgID", s.ParseRequest(createOrganisationRequest{}), s.CreateOrganisation())

	admin := auth.Use(s.IsAdmin())
	admin.POST("/_matrix/client/r0/organisation/:orgID/invite", s.ParseRequest(inviteUserRequest{}), s.InviteUser())

	return &s
}

// Server is a gin handler generator.
type Server struct {
	server   *http.Server
	db       *gorp.DbMap
	matrix   *gomatrix.Client
	mailer   Mailer
	verifier verifier
}

// Run starts the Server.
func (s *Server) Run() error { return s.server.ListenAndServe() }

// Shutdown closes the server
func (s *Server) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }

// ParseRequest parses the request into a v element, that must be a pointer.
func (s *Server) ParseRequest(v interface{}) gin.HandlerFunc {
	base := reflect.TypeOf(v)
	if base.Kind() != reflect.Struct {
		panic("interface must be a struct")
	}
	return gin.HandlerFunc(func(c *gin.Context) {
		defer c.Request.Body.Close()
		v := reflect.New(base).Interface()
		log.Printf("Parsing request in %T", v)
		if err := c.BindJSON(v); err != nil {
			ErrBadJSON.abort(c)
			return
		}
		c.Set("request", v)
	})
}

func closeTransaction(tx *gorp.Transaction, err *error) {
	if err := *err; err != nil {
		if err := tx.Rollback(); err != nil {
			log.Println("Rollback error:", err)
		}
	} else {
		if err := tx.Commit(); err != nil {
			log.Println("Commit error:", err)
		}
	}
}

type verifier []byte

func (v verifier) Hash(orgID string, email string) string {
	h := hmac.New(sha256.New, []byte(v))
	h.Write([]byte(email))
	return hex.EncodeToString(h.Sum([]byte(orgID)))
}
