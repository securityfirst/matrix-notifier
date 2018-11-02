package server

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid"
	"github.com/securityfirst/matrix-notifier/database"
)

// ViewNotifications returns a list of notifications for the current user.
func (s *Server) ViewNotifications() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		var since time.Time
		if s := c.Query("since"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				ErrBadTimestamp.abort(c)
				return
			}
			since = t
		}
		list, err := database.ListNotifications(s.db, c.MustGet("user").(string), since.Unix(),
			NPanic, NAnnouncement, NQuestion, NAnswer, NPool)
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		c.JSON(http.StatusOK, list)
	})
}

type createNotificationRequest struct {
	Destination string            `json:"destination,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	Type        string            `json:"type,omitempty"`
	Content     *database.Content `json:"content,omitempty"`
}

// CreateNotification creates a new Notification.
func (s *Server) CreateNotification() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		req := c.MustGet("request").(*createNotificationRequest)
		user, org := c.MustGet("user").(string), req.Destination
		ou, err := database.FindOrganisationUserByUserOrg(s.db, user, org)
		if err != nil {
			if err == sql.ErrNoRows {
				ErrUnauthorized.abort(c)
			} else {
				ErrUnknown.with(err).abort(c)
			}
			return
		}
		// no questions for simple users
		if ou.Level == LvlUser {
			for _, t := range []string{NBroadcast, NAnnouncement, NQuestion, NPool} {
				if req.Type == t {
					ErrUnauthorized.abort(c)
					return
				}
			}
		}
		n := database.Notification{
			ID:          ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String(),
			UserID:      user,
			Destination: req.Destination,
			Priority:    req.Priority,
			CreatedAt:   time.Now().Unix(),
			Type:        req.Type,
			Content:     req.Content,
		}
		if err := s.validateNotification(&n); err != nil {
			ErrMissingQuestion.abort(c)
			return
		}
		if err := database.Create(s.db, &n); err != nil {
			ErrUnknown.with(err).abort(c)
		}
		c.Status(http.StatusCreated)
	})
}

func (s *Server) validateNotification(n *database.Notification) error {
	switch n.Type {
	case NAnswer:
		if n.Content == nil || n.Content.QuestionID == "" {
			return ErrMissingQuestion
		}
		var q database.Notification
		if _, err := database.Get(s.db, &q, n.Content.QuestionID); err != nil {
			if err == sql.ErrNoRows {
				return ErrQuestionNotFound
			}
			return ErrUnknown.with(err)
		}
		if q.Type != NQuestion || q.Destination != n.Destination {
			return ErrQuestionNotFound
		}
	case NVote:
		if n.Content == nil || n.Content.PoolID == "" {
			return ErrMissingPool
		}
		var q database.Notification
		if _, err := database.Get(s.db, &q, n.Content.PoolID); err != nil {
			if err == sql.ErrNoRows {
				return ErrPoolNotFound
			}
			return ErrUnknown.with(err)
		}
		if q.Type != NPool || q.Destination != n.Destination {
			return ErrPoolNotFound
		}
	}
	return nil
}

// ReadNotifications marks one/all Notification as read.
func (s *Server) ReadNotifications() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		err := database.MarkAsRead(s.db, c.MustGet("user").(string), c.Param("notID"), time.Now().Unix(),
			NPanic, NAnnouncement, NQuestion, NAnswer, NPool)
		switch {
		case err == nil:
			c.Status(http.StatusOK)
		case database.IsDuplicate(err):
			c.Status(http.StatusNoContent)
		case err == sql.ErrNoRows:
			ErrNotificationNotFound.abort(c)
		default:
			ErrUnknown.with(err).abort(c)
		}
	})
}
