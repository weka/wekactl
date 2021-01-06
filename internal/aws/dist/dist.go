package dist

import (
	"errors"
	"fmt"
	"wekactl/internal/env"
)

var LambdasSource = map[string]string{}
var LambdasID string

type LambdaPackage string

const (
	ScaleIn LambdaPackage = "scale_in_lambda.zip"
	WekaCtl LambdaPackage = "wekactl-aws-lambdas.zip"
)

func GetLambdaBucket() (bucket string, err error) {
	bucket, ok := LambdasSource[env.Config.Region]
	if !ok {
		return "", errors.New(fmt.Sprintf("bucket not defined for %s", env.Config.Region))
	}
	return
}

func GetLambdaLocation(lambdaPackage LambdaPackage) (location string, err error) {
	if LambdasID == "" {
		return "", errors.New("lambda ID not defined")
	}
	bucket, err := GetLambdaBucket()
	if err != nil {
		return "", err
	}
	location = fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s/%s", env.Config.Region, bucket, LambdasID, string(lambdaPackage))
	return
}
