package dist

import (
	"errors"
	"fmt"
)

var LambdasSource = map[string]string{}
var LambdasID string

type LambdaPackage string

const (
	ScaleIn LambdaPackage = "scale_in_lambda.zip"
	WekaCtl LambdaPackage = "wekactl-aws-lambdas.zip"
)

func GetLambdaLocation(lambdaPackage LambdaPackage, region string) (location string, err error) {
	if LambdasID == "" {
		return "", errors.New("lambda ID not defined")
	}
	bucket, ok := LambdasSource[region]
	if !ok {
		return "", errors.New(fmt.Sprintf("bucket not defined for %s", region))
	}
	location = fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s/%s", region, bucket, LambdasID, string(lambdaPackage))
	return
}
