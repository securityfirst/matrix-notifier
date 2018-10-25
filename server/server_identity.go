package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"time"

	"gopkg.in/gorp.v2"

	"github.com/gin-gonic/gin"
	"github.com/matrix-org/gomatrix"

	"github.com/securityfirst/matrix-notifier/database"
)

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
