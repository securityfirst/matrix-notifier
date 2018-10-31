package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
		list, err := database.ListNotifications(s.db, c.MustGet("user").(string), since.Unix())
		if err != nil {
			ErrUnknown.with(err).abort(c)
			return
		}
		c.JSON(http.StatusOK, list)
	})
}

// PostNotifications creates a new Notification.
func (s *Server) PostNotifications() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		//TODO implement!
	})
}

// ReadNotifications marks one/all Notification as read.
func (s *Server) ReadNotifications() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		//TODO implement!
	})
}
