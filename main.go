package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/rs/zerolog/log"
)

var config struct {
	apiUserToken   string
	clusterID      string
	jwksEndpoint   string
	organizationID string
	port           string
}

func main() {
	config.apiUserToken = os.Getenv("API_USER_TOKEN")
	if config.apiUserToken == "" {
		log.Fatal().Msg("API_USER_TOKEN is required")
	}
	config.clusterID = os.Getenv("CLUSTER_ID")
	if config.clusterID == "" {
		log.Fatal().Msg("CLUSTER_ID is required")
	}
	config.jwksEndpoint = os.Getenv("JWKS_ENDPOINT")
	if config.jwksEndpoint == "" {
		log.Fatal().Msg("JWKS_ENDPOINT is required")
	}
	config.organizationID = os.Getenv("ORGANIZATION_ID")
	if config.organizationID == "" {
		log.Fatal().Msg("ORGANIZATION_ID is required")
	}
	config.port = os.Getenv("PORT")
	if config.port == "" {
		config.port = "8000"
	}

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func run(ctx context.Context) error {
	return serve(ctx)
}
