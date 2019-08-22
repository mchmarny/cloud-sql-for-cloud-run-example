package main

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/gin-gonic/gin"

	ev "github.com/mchmarny/gcputil/env"
	pr "github.com/mchmarny/gcputil/project"
)

const (
	defaultPort      = "8080"
	portVariableName = "PORT"
)

var (
	logger     = log.New(os.Stdout, "", 0)
	projectID  = pr.GetIDOrFail()
	connString = ev.MustGetEnvVar("DSN", "")
	certBucket = ev.MustGetEnvVar("CERTS", "")
	initError  error
)

func main() {

	ctx := context.Background()
	if err := initData(ctx); err != nil {
		logger.Printf("Error while initializing data: %v", err)
		initError = err
	}
	defer closeData(ctx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// routes
	r.GET("/", defaultRequestHandler)
	r.GET("/health", healthHandler)

	// api
	v1 := r.Group("/v1")
	{
		v1.GET("/test", apiRequestHandler)
	}

	// server
	port := ev.MustGetEnvVar(portVariableName, defaultPort)
	hostPost := net.JoinHostPort("0.0.0.0", port)
	logger.Printf("Server starting: %s \n", hostPost)
	if err := r.Run(hostPost); err != nil {
		logger.Fatal(err)
	}
}
