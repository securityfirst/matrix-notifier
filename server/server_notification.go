package server

import "github.com/gin-gonic/gin"

// ViewNotifications returns a list of notifications for the current user.
func (s *Server) ViewNotifications() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		//TODO implement!
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
