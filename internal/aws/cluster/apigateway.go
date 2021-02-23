package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/cluster"
)

const joinApiVersion = "v1"

type ApiGateway struct {
	RestApiGateway apigateway.RestApiGateway
	HostGroupInfo  hostgroups.HostGroupInfo
	Backend        Lambda
	TableName      string
	Version        string
	ASGName        string
}

func (a *ApiGateway) Tags() interface{} {
	return common.GetHostGroupTags(a.HostGroupInfo, a.TargetVersion()).AsStringRefs()
}

func (a *ApiGateway) SubResources() []cluster.Resource {
	return []cluster.Resource{&a.Backend}
}

func (a *ApiGateway) Init() {
	log.Debug().Msgf("Initializing hostgroup %s api gateway ...", string(a.HostGroupInfo.Name))
	a.Backend.TableName = a.TableName
	a.Backend.HostGroupInfo = a.HostGroupInfo
	a.Backend.Permissions = iam.GetJoinAndFetchLambdaPolicy()
	a.Backend.Type = lambdas.LambdaJoin
	a.Backend.Init()
}

func (a *ApiGateway) ResourceName() string {
	return common.GenerateResourceName(a.HostGroupInfo.ClusterName, a.HostGroupInfo.Name)
}

func (a *ApiGateway) Fetch() error {
	version, err := apigateway.GetRestApiGatewayVersion(a.ResourceName())
	if err != nil {
		return err
	}
	a.Version = version
	return nil
}

func (a *ApiGateway) DeployedVersion() string {
	return a.Version
}

func (a *ApiGateway) TargetVersion() string {
	return joinApiVersion
}

func (a *ApiGateway) Delete() error {
	err := a.Backend.Delete()
	if err != nil {
		return err
	}
	return apigateway.DeleteRestApiGateway(a.ResourceName())
}

func (a *ApiGateway) Create() error {
	restApiGateway, err := apigateway.CreateJoinApi(a.Tags().(common.TagsRefsValues), a.Backend.Arn, a.Backend.ResourceName(), a.ResourceName())
	if err != nil {
		return err
	}
	a.RestApiGateway = restApiGateway
	return nil
}

func (a *ApiGateway) Update() error {
	// DO actual update of this resource
	return nil
}
