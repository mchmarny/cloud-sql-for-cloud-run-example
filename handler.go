package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func healthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

func apiRequestHandler(c *gin.Context) {

	// TODO: Normalize this across sessions
	sessionID := newResponseID()

	resp := &ResponseObject{
		ID: sessionID,
		Ts: time.Now().UTC().String(),
	}

	if initError != nil {
		resp.Info = initError.Error()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	count, err := countSession(c.Request.Context(), sessionID)
	if err != nil {
		logger.Printf("Error while quering DB: %v", err)
		resp.Info = err.Error()
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp.Info = fmt.Sprintf("Success - saved %d records", count)
	c.JSON(http.StatusOK, resp)

}

func defaultRequestHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "See /v1/test",
		"status":  http.StatusOK,
	})
}

// ResponseObject represents body of the request response
type ResponseObject struct {
	ID   string `json:"request_id"`
	Ts   string `json:"request_on"`
	Info string `json:"info,omitempty"`
}

func newResponseID() string {
	id, err := uuid.NewRandom()
	if err != nil {
		logger.Fatalf("Error while getting id: %v\n", err)
	}
	return id.String()
}
