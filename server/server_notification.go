package server

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/securityfirst/matrix-notifier/database"
)

// ViewNotifications returns a list of notifications for the current user.
func (s *Server) ViewNotifications() gin.HandlerFunc {
	return handler(func(c *gin.Context) error {
		var since time.Time
		if s := c.Query("since"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return ErrBadTimestamp
			}
			since = t
		}
		rooms, err := getRooms(c)
		if err != nil {
			return err
		}
		levels := make(map[string]int, len(rooms))
		for _, id := range rooms {
			o, err := getOrgLevel(c, &database.Org{RoomID: id})
			if err != nil {
				return err
			}
			levels[id] = o.Level
		}
		list, err := database.ListNotifications(s.db, since, getUser(c), levels, rulesView)
		if err != nil {
			return err
		}
		c.JSON(http.StatusOK, list)
		return nil
	})
}

func contains(s []string, v string) bool {
	for _, a := range s {
		if v == a {
			return true
		}
	}
	return false
}

// CreateNotification creates a new Notification.
func (s *Server) CreateNotification() gin.HandlerFunc {
	return handler(func(c *gin.Context) error {
		n := c.MustGet("request").(*database.Notification)
		rooms, err := getRooms(c)
		if err != nil {
			return err
		}
		if !contains(rooms, n.RoomID) {
			return ErrUnknownOrg
		}
		v, err := s.db.Get(database.Org{}, n.RoomID)
		if err != nil {
			return err
		}
		orgLvl, err := getOrgLevel(c, v.(*database.Org))
		if err != nil {
			return err
		}
		if rulesCreate[n.RoomID] > orgLvl.Level {
			return ErrUnauthorized
		}
		n.ID, n.UserID, n.CreatedAt = newULID(), getUser(c), time.Now()
		if err := s.validateNotification(n); err != nil {
			return err
		}
		if err := database.Create(s.db, n); err != nil {
			return err
		}
		c.Status(http.StatusCreated)
		return nil
	})
}

func (s *Server) validateNotification(n *database.Notification) error {
	if n.Type != NAnswer && n.Type != NVote {
		return nil
	}
	if n.Content == nil || n.Content.RefID == "" {
		return ErrMissingReference
	}
	q := database.Notification{}
	if _, err := database.Get(s.db, &q, n.Content.RefID); err != nil {
		if err == sql.ErrNoRows {
			return ErrReferenceNotFound
		}
		return err
	}
	if q.RoomID != n.RoomID ||
		q.Type == NQuestion && n.Type != NAnswer ||
		q.Type == NPoll && n.Type != NVote {
		return ErrReferenceNotFound
	}
	return nil
}

// ReadNotifications updates the read notificaitons.
func (s *Server) ReadNotifications() gin.HandlerFunc {
	return handler(func(c *gin.Context) error {
		return database.MarkAsRead(s.db, getUser(c), time.Now())
	})
}
