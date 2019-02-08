package server

import (
	"net/http"

	"github.com/matrix-org/gomatrix"

	"github.com/gin-gonic/gin"
	"github.com/securityfirst/matrix-notifier/database"
)

// ListOrgs returns Orgs for the current User.
func (s *Server) ListOrgs() gin.HandlerFunc {
	return handler(func(c *gin.Context) error {
		rooms, err := getRooms(c)
		if err != nil {
			return err
		}
		orgs, err := database.ListOrgs(s.db, rooms...)
		if err != nil {
			return err
		}
		list := make([]*orgLevel, len(orgs))
		for i := range orgs {
			list[i], err = getOrgLevel(c, orgs[i])
			if err != nil {
				return err
			}
		}
		c.JSON(http.StatusOK, list)
		return nil
	})
}

// GetOrgByName returns an Org.
func (s *Server) GetOrgByName(name string) gin.HandlerFunc {
	return handler(func(c *gin.Context) error {
		rooms, err := getRooms(c)
		if err != nil {
			return err
		}
		org, err := database.GetOrgByName(s.db, c.Param(name), rooms...)
		if err != nil {
			return err
		}
		orgLvl, err := getOrgLevel(c, org)
		if err != nil {
			return err
		}
		c.JSON(http.StatusOK, orgLvl)
		return nil
	})
}

// CreateOrg returns an Org.
func (s *Server) CreateOrg() gin.HandlerFunc {
	return handler(func(c *gin.Context) error {
		req, client := getRequest(c).(*database.Org), getClient(c)
		room, err := client.CreateRoom(&gomatrix.ReqCreateRoom{
			Visibility:    "private",
			Name:          req.Name,
			RoomAliasName: req.Name,
		})
		if err != nil {
			return err
		}
		req.RoomID = room.RoomID
		defer func(err *error) {
			if *err != nil {
				client.ForgetRoom(room.RoomID)
			}
		}(&err)
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		defer closeTransaction(tx, &err)
		if err = database.Create(tx, req); err != nil {
			if database.IsDuplicate(err) {
				return ErrOrgExists
			}
			return err
		}
		c.Status(http.StatusCreated)
		return nil
	})
}
