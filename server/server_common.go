package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"github.com/oklog/ulid"

	"gopkg.in/gorp.v2"

	"github.com/gin-gonic/gin"
	"github.com/matrix-org/gomatrix"
)

var entropy = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

// List of Notification Types
const (
	NPanic        = "panic"        // Panic, sent by user, seen by user
	NBroadcast    = "broadcast"    // Broadcast, sent by admin, seen by admin
	NAnnouncement = "announcement" // Announcement, sent by admin, seen by user
	NQuestion     = "question"     // Question, sent by admin, seen by user
	NAnswer       = "answer"       // Answer, sent by user, seen by user, requires Question
	NPool         = "pool  "       // Pool, sent by admin, seen by user
	NVote         = "vote"         // Vote, sent by user, seen by admin, requires Pool
)

// List of Levels
const (
	LvlUser = iota
	LvlAdmin
	LvlOwner
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
	auth.GET("/_matrix/client/r0/notification", s.ViewNotifications())
	auth.POST("/_matrix/client/r0/notification", s.ParseRequest(createNotificationRequest{}), s.CreateNotification())
	auth.PATCH("/_matrix/client/r0/notification/:notID/read", s.ReadNotifications())

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
