package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/matrix-org/gomatrix"
)

// List of ErrorReponse
var (
	ErrBadJSON              = ErrorResponse{http.StatusBadRequest, "BAD_JSON", "Please provide a valid JSON"}
	ErrBadTimestamp         = ErrorResponse{http.StatusBadRequest, "BAD_TIMESTAMP", "Invalid RFC3339 timestamp"}
	ErrBadEmail             = ErrorResponse{http.StatusBadRequest, "BAD_EMAIL", "Please provide a valid email"}
	ErrMissingReference     = ErrorResponse{http.StatusBadRequest, "BAD_REFERENCE", "Please specify a reference ID"}
	ErrReferenceNotFound    = ErrorResponse{http.StatusNotFound, "UNKNOWN_REFERENCE", "Reference not found"}
	ErrNotificationNotFound = ErrorResponse{http.StatusNotFound, "UNKNOWN_NOTIFICATION", "Notification not found"}
	ErrMissingToken         = ErrorResponse{http.StatusUnauthorized, "M_MISSING_TOKEN", "Missing access token"}
	ErrUnknownToken         = ErrorResponse{http.StatusUnauthorized, "UNKNOWN_TOKEN", "Unknown Access Token"}
	ErrUnauthorized         = ErrorResponse{http.StatusUnauthorized, "M_UNAUTHORIZED", "Not allowed"}
	ErrUnknownOrg           = ErrorResponse{http.StatusBadRequest, "UNKNOWN_ORG", "Unknown Org"}
	ErrOrgExists            = ErrorResponse{http.StatusConflict, "ORG_EXISTS", "Org name already exists"}
)

var (
	errAdmin = errors.New("must be admin")
)

// ErrorResponse is Matrix Error response.
type ErrorResponse struct {
	Status int    `json:"-"`
	Code   string `json:"errcode,omitempty"`
	Err    string `json:"error,omitempty"`
}

func (e ErrorResponse) with(err error) ErrorResponse {
	if e, ok := err.(gomatrix.RespError); ok {
		return ErrorResponse{
			Status: http.StatusBadRequest,
			Code:   e.ErrCode,
			Err:    e.Err,
		}
	}
	e.Err = err.Error()
	return e
}

func (e ErrorResponse) Error() string { return fmt.Sprintf("%s (%s)", e.Code, e.Err) }

func handler(fn func(c *gin.Context) error) func(c *gin.Context) {
	return func(c *gin.Context) {
		if err := fn(c); err != nil {
			e, ok := err.(ErrorResponse)
			if !ok {
				for i := range []int{0, 1, 2, 3} {
					_, file, line, _ := runtime.Caller(i)
					idx := 0
					log.Printf("[Error] %s:%d - %s", file[idx:], line, err)
				}
				e = ErrorResponse{http.StatusInternalServerError, "UNKNOWN", err.Error()}
			}
			c.AbortWithStatusJSON(e.Status, e)
		}
	}
}
