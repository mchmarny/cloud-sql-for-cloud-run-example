package main

import (
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

	countSession(c.Request.Context(), sessionID)

	resp := &ResponseObject{
		ID:      sessionID,
		Ts:      time.Now().UTC().String(),
		Bucket:  certBucket,
		Conn:    connString,
		KeyRing: kmsKeyRing,
	}

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
	ID      string `json:"id"`
	Ts      string `json:"ts"`
	Bucket  string `json:"conf"`
	Conn    string `json:"conn"`
	KeyRing string `json:"keyring"`
}

func newResponseID() string {
	id, err := uuid.NewRandom()
	if err != nil {
		logger.Fatalf("Error while getting id: %v\n", err)
	}
	return id.String()
}
