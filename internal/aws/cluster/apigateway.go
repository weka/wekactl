package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/apigateway"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
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
	version, err := db.GetResourceVersion(a.TableName, "apigateway", "", a.HostGroupInfo.Name)
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
	return joinApiVersion + a.Backend.TargetVersion()
}

func (a *ApiGateway) Delete() error {
	err := a.Backend.Delete()
	if err != nil {
		return err
	}
	return apigateway.DeleteRestApiGateway(a.ResourceName())
}

func (a *ApiGateway) Create() error {
	err := cluster.EnsureResource(&a.Backend)
	if err != nil {
		return err
	}
	restApiGateway, err := apigateway.CreateJoinApi(a.HostGroupInfo, a.Backend.Type, a.Backend.Arn, a.Backend.ResourceName(), a.ResourceName())
	if err != nil {
		return err
	}
	a.RestApiGateway = restApiGateway
	return db.SaveResourceVersion(a.TableName, "apigateway", "", a.HostGroupInfo.Name, a.TargetVersion())
}

func (a *ApiGateway) Update() error {
	if a.DeployedVersion() == a.TargetVersion() {
		return nil
	}
	err := a.Backend.Update()
	if err != nil {
		return err
	}
	return nil
}
