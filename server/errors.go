package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// List of ErrorReponse
var (
	ErrUnknown          = ErrorResponse{http.StatusInternalServerError, "UNKNOWN", "Internal server error"}
	ErrBadJSON          = ErrorResponse{http.StatusBadRequest, "BAD_JSON", "Please provide a valid JSON"}
	ErrBadTimestamp     = ErrorResponse{http.StatusBadRequest, "BAD_TIMESTAMP", "Invalid RFC3339 timestamp."}
	ErrBadEmail         = ErrorResponse{http.StatusBadRequest, "BAD_EMAIL", "Please provide a valid email"}
	ErrMissingQuestion  = ErrorResponse{http.StatusBadRequest, "BAD_QUESTION", "Please specify a question ID."}
	ErrQuestionNotFound = ErrorResponse{http.StatusNotFound, "UNKNOWN_QUESTION", "Question not found."}
	ErrMissingToken     = ErrorResponse{http.StatusUnauthorized, "M_MISSING_TOKEN", "Missing access token."}
	ErrUnknownToken     = ErrorResponse{http.StatusUnauthorized, "UNKNOWN_TOKEN", "Unknown Access Token."}
	ErrUnauthorized     = ErrorResponse{http.StatusUnauthorized, "M_UNAUTHORIZED", "Not allowed"}
	ErrUnknownOrg       = ErrorResponse{http.StatusBadRequest, "UNKNOWN_ORG", "Unknown Organisation."}
)

var (
	errAdmin = errors.New("Must be admin.")
)

// ErrorResponse is Matrix Error response.
type ErrorResponse struct {
	Status int    `json:"-"`
	Code   string `json:"errcode,omitempty"`
	Err    string `json:"error,omitempty"`
}

func (e ErrorResponse) with(err error) ErrorResponse {
	if e == ErrUnknown {
		_, file, line, _ := runtime.Caller(1)
		idx := strings.Index(file, "/server/")
		log.Printf("[Error] %s:%d - %s", file[idx:], line, err)
	}
	e.Err = err.Error()
	return e
}

func (e ErrorResponse) Error() string { return fmt.Sprintf("%s (%s)", e.Code, e.Err) }

func (e ErrorResponse) abort(c *gin.Context) { c.AbortWithStatusJSON(e.Status, e) }
