package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/env"
)

func joinHandler() (events.APIGatewayProxyResponse, error) {
	result, err := lambdas.GetJoinParams(
		os.Getenv("ASG_NAME"),
		os.Getenv("TABLE_NAME"),
	)
	if err != nil {
		result = err.Error()
	}
	return events.APIGatewayProxyResponse{Body: result, StatusCode: 200}, nil

}

func main() {
	env.Config.Region = os.Getenv("REGION")
	if os.Getenv("LAMBDA") == "join" {
		lambda.Start(joinHandler)
	}
}
