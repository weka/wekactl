package cleaner

import (
	"github.com/aws/aws-sdk-go/service/lambda"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type Lambda struct {
	Lambdas []*lambda.FunctionConfiguration
}

func (l *Lambda) Fetch(clusterName cluster.ClusterName) error {
	lambdaConfigurations, err := lambdas.GetClusterLambdas(clusterName)
	if err != nil {
		return err
	}
	l.Lambdas = lambdaConfigurations
	return nil
}

func (l *Lambda) Delete() error {
	return lambdas.DeleteLambdas(l.Lambdas)
}

func (l *Lambda) Print() {
	logging.UserInfo("Lambdas:")
	for _, lambdaConfiguration := range l.Lambdas {
		logging.UserInfo("\t- %s", *lambdaConfiguration.FunctionName)
	}
}
