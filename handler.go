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

	msg := c.Param("msg")
	logger.Printf("Message: %s", msg)
	if msg == "" {
		logger.Println("Error on nil msg parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Null Argument",
			"status":  http.StatusBadRequest,
		})
		return
	}

	resp := &ResponseObject{
		ID:      newID(),
		Ts:      time.Now().UTC().String(),
		Release: release,
		Message: msg,
	}

	c.JSON(http.StatusOK, resp)

}

func defaultRequestHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Try the API at /v1/message/:msg",
		"status":  http.StatusOK,
	})
}

// ResponseObject represents body of the request response
type ResponseObject struct {
	ID      string `json:"id"`
	Ts      string `json:"ts"`
	Release string `json:"rel"`
	Message string `json:"msg,omitempty"`
}

func newID() string {
	id, err := uuid.NewRandom()
	if err != nil {
		logger.Fatalf("Error while getting id: %v\n", err)
	}
	return id.String()
}
