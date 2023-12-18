package cluster

import (
	"github.com/aws/aws-sdk-go/aws"
	"strconv"
	"strings"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/dist"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/cluster"
	strings2 "wekactl/internal/lib/strings"

	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/rs/zerolog/log"
)

type Lambda struct {
	Arn                 string
	TableName           string
	Version             string
	ASGName             string
	Type                lambdas.LambdaType
	Profile             IamProfile
	VPCConfig           lambda.VpcConfig
	HostGroupInfo       common.HostGroupInfo
	Permissions         iam.PolicyDocument
	UseDynamoDBEndpoint bool
}

func (l *Lambda) Tags() cluster.Tags {
	return GetHostGroupResourceTags(l.HostGroupInfo, l.TargetVersion())
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

	if l.Profile.Arn == "" {
		profileArn, err := iam.GetIamRoleArn(l.HostGroupInfo.ClusterName, l.Profile.resourceNameBase())
		if err != nil {
			return err
		}
		l.Profile.Arn = profileArn
	}

	if l.Version != "" {
		arn, err := lambdas.GetLambdaRoleArn(l.ResourceName())
		if err != nil {
			return err
		}
		if arn != l.Profile.Arn {
			l.Version = l.Version + "#" // just to make it different from TargetVersion so we will enter Update flow
		}
	}

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

func (l *Lambda) Create(tags cluster.Tags) (err error) {
	functionConfiguration, err := lambdas.CreateLambda(
		tags.AsStringRefs(), l.Type, l.ResourceName(), l.Profile.Arn, l.ASGName, l.TableName, l.HostGroupInfo, l.VPCConfig, l.UseDynamoDBEndpoint)
	if err != nil {
		return
	}
	l.Arn = *functionConfiguration.FunctionArn

	return nil
}

func (l *Lambda) Update(tags cluster.Tags) error {
	// check if runtime is different
	info, err := lambdas.GetLambdaRuntime(l.ResourceName())
	if err != nil {
		return err
	}
	if info.Runtime != lambdas.LambdaRuntimeDefault || info.HandlerName != lambdas.LambdaHandlerName {
		err := lambdas.UpdateLambdaRuntime(l.ResourceName(), lambdas.LambdaRuntimeDefault, lambdas.LambdaHandlerName)
		if err != nil {
			return err
		}
	}
	if info.Arch != lambdas.LambdaArchDefault {
		err := lambdas.UpdateLambdaArchitecture(l.ResourceName(), lambdas.LambdaArchDefault)
		if err != nil {
			return err
		}
	}
	useDynamoDBEndpoint, err := strconv.ParseBool(*info.EnvironmentVariables["USE_DYNAMODB_ENDPOINT"])
	if err != nil {
		return err
	}
	if useDynamoDBEndpoint != l.UseDynamoDBEndpoint {
		info.EnvironmentVariables["USE_DYNAMODB_ENDPOINT"] = aws.String(strconv.FormatBool(l.UseDynamoDBEndpoint))
		err := lambdas.UpdateLambdaEnvironmentVariable(l.ResourceName(), info.EnvironmentVariables)
		if err != nil {
			return err
		}
	}

	if strings.HasSuffix(l.DeployedVersion(), "#") {
		err := lambdas.UpdateLambdaRole(l.ResourceName(), l.Profile.Arn)
		if err != nil {
			return err
		}
	}
	if l.DeployedVersion() != l.TargetVersion() && l.DeployedVersion() != l.TargetVersion()+"#" {
		return lambdas.UpdateLambdaHandler(l.ResourceName(), cluster.GetResourceVersionTag(l.TargetVersion()).AsStringRefs())
	}
	return nil
}
