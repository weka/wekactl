package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/weka/go-cloud-lib/scale_down"
	"os"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/aws/lambdas/protocol"
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

func fetchHandler() (protocol.HostGroupInfoResponse, error) {
	result, err := lambdas.GetFetchDataParams(
		os.Getenv("CLUSTER_NAME"),
		os.Getenv("ASG_NAME"),
		os.Getenv("TABLE_NAME"),
		os.Getenv("ROLE"),
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
		lambda.Start(scale_down.ScaleDown)
	case "terminate":
		lambda.Start(terminate.Handler)
	case "transient":
		lambda.Start(transient.Handler)
	default:
		lambda.Start(func() error { return errors.New("unsupported lambda command") })
	}
}
