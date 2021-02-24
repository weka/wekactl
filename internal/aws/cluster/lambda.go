package cluster

import (
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/dist"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/cluster"
	strings2 "wekactl/internal/lib/strings"
)

type Lambda struct {
	Arn           string
	TableName     string
	Version       string
	ASGName       string
	Type          lambdas.LambdaType
	Profile       IamProfile
	VPCConfig     lambda.VpcConfig
	HostGroupInfo hostgroups.HostGroupInfo
	Permissions   iam.PolicyDocument
}

func (l *Lambda) Tags() interface{} {
	return common.GetHostGroupTags(l.HostGroupInfo, l.TargetVersion()).AsStringRefs()
}

func (l *Lambda) SubResources() []cluster.Resource {
	return []cluster.Resource{&l.Profile}
}

func (l *Lambda) ResourceName() string {
	n := strings.Join([]string{"wekactl", string(l.HostGroupInfo.ClusterName), string(l.Type), string(l.HostGroupInfo.Name)}, "-")
	return strings2.ElfHashSuffixed(n, 64)
}

func (l *Lambda) Fetch() error {
	version, err := lambdas.GetLambdaVersion(l.ResourceName())
	if err != nil {
		return err
	}
	l.Version = version
	return nil
}

func (l *Lambda) Init() {
	log.Debug().Msgf("Initializing hostgroup %s %s lambda ...", string(l.HostGroupInfo.Name), string(l.Type))
	l.Profile.Name = string(l.Type)
	l.Profile.PolicyName = l.ResourceName()
	l.Profile.TableName = l.TableName
	l.Profile.AssumeRolePolicy = iam.GetLambdaAssumeRolePolicy()
	l.Profile.HostGroupInfo = l.HostGroupInfo
	l.Profile.Policy = l.Permissions
	l.Profile.Init()
}

func (l *Lambda) DeployedVersion() string {
	return l.Version
}

func (l *Lambda) TargetVersion() string {
	return dist.LambdasID
}

func (l *Lambda) Delete() error {
	return lambdas.DeleteLambda(l.ResourceName())
}

func (l *Lambda) Create() (err error) {
	functionConfiguration, err := lambdas.CreateLambda(
		l.Tags().(common.TagsRefsValues), l.Type, l.ResourceName(), l.Profile.Arn, l.ASGName, l.TableName, l.HostGroupInfo.Role, l.VPCConfig)
	if err != nil {
		return
	}
	l.Arn = *functionConfiguration.FunctionArn

	return nil
}

func (l *Lambda) Update() error {
	return lambdas.UpdateLambdaHandler(l.ResourceName())
}
