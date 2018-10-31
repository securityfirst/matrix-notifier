package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/securityfirst/matrix-notifier/database"
)

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
			if err := s.inviteUser(tx, org.ID, req.Admin, LvlOwner); err != nil {
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
