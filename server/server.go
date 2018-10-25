package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"reflect"
	"time"

	"gopkg.in/gorp.v2"

	"github.com/gin-gonic/gin"
	"github.com/matrix-org/gomatrix"

	"github.com/securityfirst/matrix-notifier/database"
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

// IsAdmin checks that the current User is admin for the Organisation.
func (s *Server) IsAdmin() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		ou, err := database.FindOrganisationUserByUserOrg(s.db, c.MustGet("user").(string), c.Param("orgID"))
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		if ou.Level == database.LvlUser {
			ErrUnauthorized.with(errAdmin).abort(c)
			return
		}
		c.Set("organisation_user", ou)

	})
}

// Authenticate saves the user associated with the token provided in query.
func (s *Server) Authenticate() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		u, err := s.authenticate(c)
		if err != nil {
			err.(ErrorResponse).abort(c)
			return
		}
		c.Set("user", u)
	})
}

func (s *Server) authenticate(c *gin.Context) (string, error) {
	token := c.Query("access_token")
	if token == "" {
		return "", ErrMissingToken
	}
	user, err := database.FindUserByToken(s.db, token)
	if err != nil {
		return "", ErrUnknown.with(err)
	}
	if user == "" {
		return "", ErrUnknownToken
	}
	return user, nil
}

// ListUserOrganisations returns Organisations for the user associated with provided token.
func (s *Server) ListUserOrganisations() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		orgs, err := database.ListOrganisations(s.db, c.MustGet("user").(string))
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		c.JSON(http.StatusOK, orgs)
	})
}

// GetOrganisation returns an Organisation.
func (s *Server) GetOrganisation() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		org, err := s.db.Get(&database.Organisation{}, c.Param("orgID"))
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		if org == nil {
			c.JSON(http.StatusNotFound, ErrUnknownOrg)
		}
		c.JSON(http.StatusOK, org)
	})
}

// createOrganisationRequest is the input of CreateOrganisation.
type createOrganisationRequest struct {
	Name    string `json:"name"`
	Intent  string `json:"intent"`
	Package string `json:"package"`
	Admin   string `json:"admin"`
}

// CreateOrganisation returns an Organisation.
func (s *Server) CreateOrganisation() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		req := c.MustGet("request").(*createOrganisationRequest)
		org := database.Organisation{
			ID:      c.Param("orgID"),
			Name:    req.Name,
			Package: req.Package,
			Intent:  req.Intent,
		}
		tx, err := s.db.Begin()
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		defer closeTransaction(tx, &err)
		if err = database.Create(tx, &org); err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		if req.Admin != "" {
			if err := s.inviteUser(tx, org.ID, req.Admin, database.LvlOwner); err != nil {
				err.(ErrorResponse).abort(c)
			}
		}
		c.Status(http.StatusCreated)
	})
}

// RedirectToIntent redirect to the Organisation intent.
func (s *Server) RedirectToIntent() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		ou, err := database.FindOrganisationUserByHash(s.db, c.Query("hash"))
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		org, err := database.Get(s.db, &database.Organisation{}, ou.OrganisationID)
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s://verify?hash=%s", org.(*database.Organisation).Intent, c.Query("hash")))
	})
}

// inviteUserRequest is the input of InviteUser.
type inviteUserRequest struct {
	Email string `json:"email"`
	Admin bool   `json:"admin"`
}

// InviteUser redirect to the Organisation intent.
func (s *Server) InviteUser() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		req := c.MustGet("request").(*inviteUserRequest)
		tx, err := s.db.Begin()
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		lvl := database.LvlUser
		if req.Admin {
			lvl = database.LvlAdmin
		}
		defer closeTransaction(tx, &err)
		if err = s.inviteUser(tx, c.Param("orgID"), req.Email, lvl); err != nil {
			err.(ErrorResponse).abort(c)
		}
	})
}

func (s *Server) inviteUser(tx *gorp.Transaction, orgID string, email string, lvl int) error {
	m, err := mail.ParseAddress(email)
	if err != nil {
		return ErrBadEmail
	}
	hash := s.verifier.Hash(orgID, m.Address)
	ou := database.OrganisationUser{
		OrganisationID: orgID,
		Hash:           hash,
		Level:          lvl,
	}
	if err := database.Create(tx, &ou); err != nil {
		if database.IsDuplicate(err) {
			return ErrBadEmail.with(errors.New("Invite already sent."))
		}
		return ErrUnknown.with(err)
	}
	org, err := database.Get(tx, &database.Organisation{}, orgID)
	if err != nil {
		return ErrUnknown.with(err)
	}
	data := map[string]string{
		"organisation": org.(*database.Organisation).Name,
		"secret":       ou.Hash,
		"link":         fmt.Sprintf("http://%s/_matrix/client/r0/organisation/%s/verify?hash=%s", s.server.Addr, ou.OrganisationID, ou.Hash),
	}
	if err := s.mailer.Send(m, data); err != nil {
		return ErrUnknown.with(err)
	}
	return nil
}

// createUserRequest is the input of CreateUser.
type createUserRequest struct {
	Username string
	Password string
	Email    string
	Secret   string
}

// CreateUser creates a user and adds it to an organisation.
func (s *Server) CreateUser() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		req := c.MustGet("request").(*createUserRequest)
		tx, err := s.db.Begin()
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		defer closeTransaction(tx, &err)

		user, err := s.authenticate(c)
		if err != nil && err != ErrMissingToken {
			err.(ErrorResponse).abort(c)
			return
		}
		if user != "" {
			req.Email, err = database.FindEmailByUser(s.db, user)
			if err != nil {
				ErrUnknown.with(err).abort(c)
				return
			}
		}

		hash := s.verifier.Hash(c.Param("orgID"), req.Email)
		if hash != req.Secret {
			ErrUnauthorized.with(errors.New("Secret mismatch.")).abort(c)
			return
		}
		if _, err := database.FindOrganisationUserByHash(tx, hash); err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		var resp interface{}
		if user == "" {
			r, err := s.matrix.RegisterDummy(&gomatrix.ReqRegister{
				Username: req.Username,
				Password: req.Password,
			})
			if err != nil {
				if e, ok := err.(gomatrix.RespError); ok {
					ErrorResponse{
						Status: http.StatusBadRequest,
						Code:   e.ErrCode,
						Err:    e.Err,
					}.abort(c)
					return
				}
				ErrUnknown.with(err).abort(c)
				return
			}
			user = r.UserID
			now := time.Now().UTC().Unix()
			if err := database.Create(s.db, &database.UserThreepid{
				UserID:      user,
				Medium:      "email",
				Address:     req.Email,
				ValidatedAt: now,
				AddedAt:     now,
			}); err != nil {
				ErrUnknown.with(err).abort(c)
				return
			}
			resp = r
		}
		if err := database.UpdateOrganisationUser(tx, user, hash); err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		if resp != nil {
			c.JSON(http.StatusCreated, resp)
		} else {
			c.Status(http.StatusCreated)
		}
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
