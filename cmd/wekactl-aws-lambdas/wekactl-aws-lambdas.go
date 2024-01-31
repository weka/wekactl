package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"
	"github.com/weka/go-cloud-lib/protocol"
	"os"
	"strconv"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/aws/lambdas/scale_down"
	"wekactl/internal/aws/lambdas/terminate"
	"wekactl/internal/aws/lambdas/transient"
	"wekactl/internal/env"
)

func joinHandler(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	result, err := lambdas.GetJoinParams(
		ctx,
		os.Getenv("CLUSTER_NAME"),
		os.Getenv("ASG_NAME"),
		os.Getenv("TABLE_NAME"),
		os.Getenv("ROLE"),
	)
	if err != nil {
		result = err.Error()
	}
	return events.APIGatewayProxyResponse{Body: result, StatusCode: 200}, nil
}

func fetchHandler(request protocol.FetchRequest) (protocol.HostGroupInfoResponse, error) {
	useDynamoDBEndpoint, err := strconv.ParseBool(os.Getenv("USE_DYNAMODB_ENDPOINT"))
	if err != nil {
		return protocol.HostGroupInfoResponse{}, err
	}
	fetchWekaCredentials := !useDynamoDBEndpoint || request.FetchWekaCredentials
	log.Info().Msgf("fetching data, request: %+v", request)
	result, err := lambdas.GetFetchDataParams(
		os.Getenv("CLUSTER_NAME"),
		os.Getenv("ASG_NAME"),
		os.Getenv("TABLE_NAME"),
		os.Getenv("ROLE"),
		fetchWekaCredentials,
	)
	if err != nil {
		return protocol.HostGroupInfoResponse{}, err
	}
	return result, nil
}

func main() {
	env.Config.Region = os.Getenv("REGION")
	switch lambdaType := os.Getenv("LAMBDA"); lambdaType {
	case "join":
		lambda.Start(joinHandler)
	case "fetch":
		lambda.Start(fetchHandler)
	case "scale":
		lambda.Start(scale_down.Handler)
	case "terminate":
		lambda.Start(terminate.Handler)
	case "transient":
		lambda.Start(transient.Handler)
	default:
		lambda.Start(func() error { return errors.New("unsupported lambda command") })
	}
}
