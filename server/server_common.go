package server

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"github.com/matrix-org/gomatrix"
	"github.com/securityfirst/matrix-notifier/database"

	"github.com/oklog/ulid"

	"gopkg.in/gorp.v2"

	"github.com/gin-gonic/gin"
)

var entropy = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

// List of Notification Types
const (
	NPanic        = "panic"        // Panic, sent by user, seen by user
	NBroadcast    = "broadcast"    // Broadcast, sent by admin, seen by admin
	NAnnouncement = "announcement" // Announcement, sent by admin, seen by user
	NQuestion     = "question"     // Question, sent by admin, seen by user
	NAnswer       = "answer"       // Answer, sent by user, seen by user, requires Question
	NPoll         = "poll  "       // Poll, sent by admin, seen by user
	NVote         = "vote"         // Vote, sent by user, seen by admin, requires Pool
)

// List of User Levels
const (
	LUser  = 0
	LMod   = 50
	LAdmin = 100
)

var rulesView = map[string]int{
	NPanic:        LUser,
	NBroadcast:    LMod,
	NAnnouncement: LUser,
	NQuestion:     LUser,
	NAnswer:       LUser,
	NPoll:         LUser,
	NVote:         LMod,
}

var rulesCreate = map[string]int{
	NPanic:        LUser,
	NBroadcast:    LMod,
	NAnnouncement: LMod,
	NQuestion:     LMod,
	NAnswer:       LUser,
	NPoll:         LMod,
	NVote:         LMod,
}

// List of Levels
const (
	LvlUser = iota
	LvlAdmin
	LvlOwner
)

// NewServer returns a new Server.
func NewServer(address, matrix string, db *gorp.DbMap) *Server {
	engine := gin.Default()
	s := Server{
		server: &http.Server{Addr: address, Handler: engine},
		db:     db,
		matrix: matrix,
	}
	auth := engine.Group("/_matrix/client/r0/", s.Authenticate())

	org := auth.Group("/organisation/")
	org.GET("", s.ListOrgs())
	org.POST("", s.ParseRequest(database.Org{}), s.CreateOrg())
	org.GET(":name", s.GetOrgByName("name"))

	not := auth.Group("/notification/")
	not.GET("", s.ViewNotifications())
	not.POST("", s.ParseRequest(database.Notification{}), s.CreateNotification())
	not.PATCH("", s.ReadNotifications())

	return &s
}

// Server is a gin handler generator.
type Server struct {
	server *http.Server
	db     *gorp.DbMap
	matrix string
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
	return handler(func(c *gin.Context) error {
		defer c.Request.Body.Close()
		v := reflect.New(base).Interface()
		if err := c.BindJSON(v); err != nil {
			return ErrBadJSON
		}
		c.Set(keyRequest, v)
		return nil
	})
}

const (
	keyUser    = "user"
	keyClient  = "client"
	keyRequest = "request"
)

// Authenticate saves the user associated with the token provided in query.
func (s *Server) Authenticate() gin.HandlerFunc {
	return handler(func(c *gin.Context) error {
		client, err := gomatrix.NewClient(s.matrix, "", c.Query("access_token"))
		if err != nil {
			return err
		}
		var resp struct {
			UserID string `json:"user_id"`
		}
		url := client.BuildURL("/account/whoami")
		if _, err := client.MakeRequest("GET", url, nil, &resp); err != nil {
			return err
		}

		c.Set(keyUser, resp.UserID)
		c.Set(keyClient, client)
		return nil
	})
}

func getUser(c *gin.Context) string             { return c.MustGet(keyUser).(string) }
func getClient(c *gin.Context) *gomatrix.Client { return c.MustGet(keyClient).(*gomatrix.Client) }
func getRequest(c *gin.Context) interface{}     { return c.MustGet(keyRequest) }

func getRooms(c *gin.Context) ([]string, error) {
	r, err := getClient(c).JoinedRooms()
	if err != nil {
		return nil, err
	}
	return r.JoinedRooms, nil
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

type orgLevel struct {
	*database.Org
	Level int `json:"level"`
}

func getOrgLevel(c *gin.Context, org *database.Org) (*orgLevel, error) {
	var p struct {
		Users        map[string]int `json:"users"`
		UsersDefault int            `json:"users_default"`
	}
	if err := getClient(c).StateEvent(org.RoomID, "m.room.power_levels", "", &p); err != nil {
		return nil, err
	}
	lvl, ok := p.Users[getUser(c)]
	if !ok {
		lvl = p.UsersDefault
	}
	return &orgLevel{org, lvl}, nil
}

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}
