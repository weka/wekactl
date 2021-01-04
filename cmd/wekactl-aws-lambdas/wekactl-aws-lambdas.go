package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"os"
	"wekactl/internal/aws/lambdas"
)

func joinHandler() (events.APIGatewayProxyResponse, error) {
	result, err := lambdas.GetJoinParams(
		os.Getenv("REGION"),
		os.Getenv("ASG_NAME"),
		os.Getenv("TABLE_NAME"),
		)
	if err != nil {
		result = err.Error()
	}
	return events.APIGatewayProxyResponse{Body: result, StatusCode: 200}, nil

}

func main() {
	if os.Getenv("LAMBDA") == "join" {
		lambda.Start(joinHandler)
	}
}
