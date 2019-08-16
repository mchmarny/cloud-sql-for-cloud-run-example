package main

import (
	"log"
	"net"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/thinkerou/favicon"

	ev "github.com/mchmarny/gcputil/env"
)

const (
	defaultPort      = "8080"
	portVariableName = "PORT"
)

var (
	release = ev.MustGetEnvVar("RELEASE", "v0.0.1")
	logger  = log.New(os.Stdout, "[APP] ", 0)
)

func main() {

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(favicon.New("./favicon.ico"))

	// routes
	r.GET("/", defaultRequestHandler)
	r.GET("/health", healthHandler)

	// api
	v1 := r.Group("/v1")
	{
		v1.GET("/message/:msg", apiRequestHandler)
	}

	// server
	port := ev.MustGetEnvVar(portVariableName, defaultPort)
	hostPost := net.JoinHostPort("0.0.0.0", port)
	logger.Printf("Server starting: %s \n", hostPost)
	if err := r.Run(hostPost); err != nil {
		logger.Fatal(err)
	}
}
