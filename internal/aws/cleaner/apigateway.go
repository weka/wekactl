package cleaner

import (
	"github.com/aws/aws-sdk-go/service/apigateway"
	apigateway2 "wekactl/internal/aws/apigateway"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type ApiGateway struct {
	RestApis    []*apigateway.RestApi
	ClusterName cluster.ClusterName
}

func (a *ApiGateway) Fetch() error {
	restApis, err := apigateway2.GetClusterApiGateways(a.ClusterName)
	if err != nil {
		return err
	}
	a.RestApis = restApis
	return nil
}

func (a *ApiGateway) Delete() error {
	return apigateway2.DeleteApiGateways(a.RestApis)
}

func (a *ApiGateway) Print() {
	logging.UserInfo("ApiGateways:")
	for _, restApi := range a.RestApis {
		logging.UserInfo("\t- %s", *restApi.Name)
	}
}
