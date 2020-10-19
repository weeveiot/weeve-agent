package main

import (
	"os"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/jessevdk/go-flags"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal"
	// "gitlab.com/weeve/edge-server/edge-pipeline-service/internal/constants"
)

type Options struct {
	Port    int    `long:"port" short:"p" description:"Port number" required:"true"`
	Verbose []bool `long:"verbose" short:"v" description:"Show verbose debug information"`

	// TODO: We only need this for AWS ECR integration...
	// RoleArn string `long:"role" short:"r" description:"Role Arn" required:"false"`
}

var options Options
var parser = flags.NewParser(&options, flags.Default)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	log.SetLevel(log.DebugLevel)
	log.Debug("Started logging")
}

// @title Weeve Manager API
// @version 1.0
// @description This is a weeve management api.
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Token
func main() {
	if _, err := parser.Parse(); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// TODO: Add -v flag!
	fmt.Printf("Verbosity: %v\n", options.Verbose)
	fmt.Printf("Port: %v\n", options.Port)

	// TODO: We only need this for AWS ECR integration...
	// if options.RoleArn != "" {
	// 	constants.RoleArn = options.RoleArn
	// }
	log.Info("Starting server on port ", options.Port)
	internal.HandleRequests(options.Port)
	log.Debug("Running")
}
