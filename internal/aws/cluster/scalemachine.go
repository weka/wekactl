package cluster

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/aws/scalemachine"
	"wekactl/internal/cluster"
)

const scaleMachineVersion = "v1"

type ScaleMachine struct {
	Arn             string
	TableName       string
	Version         string
	ASGName         string
	HostGroupInfo   common.HostGroupInfo
	HostGroupParams common.HostGroupParams
	fetch           Lambda
	scale           Lambda
	terminate       Lambda
	transient       Lambda
	StateMachine    scalemachine.StateMachine
	Profile         IamProfile
}

func (s *ScaleMachine) Tags() cluster.Tags {
	return GetHostGroupResourceTags(s.HostGroupInfo, s.TargetVersion())
}

func (s *ScaleMachine) SubResources() []cluster.Resource {
	return []cluster.Resource{&s.fetch, &s.scale, &s.terminate, &s.transient, &s.Profile}
}

func (s *ScaleMachine) ResourceName() string {
	return common.GenerateResourceName(s.HostGroupInfo.ClusterName, s.HostGroupInfo.Name)
}

func (s *ScaleMachine) Fetch() error {
	version, err := scalemachine.GetStateMachineVersion(s.ResourceName())
	if err != nil {
		return err
	}
	s.Version = version

	if s.Profile.Arn == "" {
		profileArn, err := iam.GetIamRoleArn(s.HostGroupInfo.ClusterName, s.Profile.resourceNameBase())
		if err != nil {
			return err
		}
		s.Profile.Arn = profileArn
	}

	if version != "" {
		scaleMachineArn, err := scalemachine.GetStateMachineArn(s.ResourceName())
		if err != nil {
			return err
		}
		s.Arn = scaleMachineArn

		arn, err := scalemachine.GetStateMachineRoleArn(s.ResourceName())
		if err != nil {
			return err
		}
		if arn != s.Profile.Arn {
			s.Version = s.Version + "#" // just to make it different from TargetVersion so we will enter Update flow
		}
	}

	if s.fetch.Arn == "" {
		backendArn, err := lambdas.GetLambdaArn(s.fetch.ResourceName())
		if err != nil {
			return err
		}
		s.fetch.Arn = backendArn
	}

	if s.scale.Arn == "" {
		backendArn, err := lambdas.GetLambdaArn(s.scale.ResourceName())
		if err != nil {
			return err
		}
		s.scale.Arn = backendArn
	}

	if s.terminate.Arn == "" {
		backendArn, err := lambdas.GetLambdaArn(s.terminate.ResourceName())
		if err != nil {
			return err
		}
		s.terminate.Arn = backendArn
	}

	if s.transient.Arn == "" {
		backendArn, err := lambdas.GetLambdaArn(s.transient.ResourceName())
		if err != nil {
			return err
		}
		s.transient.Arn = backendArn
	}

	return nil
}

func (s *ScaleMachine) DeployedVersion() string {
	return s.Version
}

func (s *ScaleMachine) TargetVersion() string {
	return scaleMachineVersion
}

func (s *ScaleMachine) Create(tags cluster.Tags) (err error) {
	stateMachineLambdasArn := scalemachine.StateMachineLambdasArn{
		Fetch:     s.fetch.Arn,
		Scale:     s.scale.Arn,
		Terminate: s.terminate.Arn,
		Transient: s.transient.Arn,
	}

	arn, err := scalemachine.CreateStateMachine(tags.AsSfn(), stateMachineLambdasArn, s.Profile.Arn, s.ResourceName())
	if err != nil {
		return
	}
	s.Arn = *arn
	return nil
}

func (s *ScaleMachine) Update(tags cluster.Tags) error {
	return scalemachine.UpdateStateMachineRoleArn(s.Arn, s.Profile.Arn)
}

func (s *ScaleMachine) Init() {
	log.Debug().Msgf("Initializing hostgroup %s state machine ...", string(s.HostGroupInfo.Name))
	s.Profile.Name = "sm"
	s.Profile.PolicyName = fmt.Sprintf("wekactl-%s-sm-%s", string(s.HostGroupInfo.ClusterName), string(s.HostGroupInfo.Name))
	s.Profile.TableName = s.TableName
	s.Profile.AssumeRolePolicy = iam.GetStateMachineAssumeRolePolicy()
	s.Profile.HostGroupInfo = s.HostGroupInfo
	s.Profile.Policy = iam.GetStateMachineRolePolicy()
	s.Profile.Init()

	vpcConfig := lambdas.GetLambdaVpcConfig(s.HostGroupParams.Subnet, s.HostGroupParams.SecurityGroupsIds)

	s.fetch.TableName = s.TableName
	s.fetch.ASGName = s.ASGName
	s.fetch.HostGroupInfo = s.HostGroupInfo
	s.fetch.Type = lambdas.LambdaFetchInfo
	s.fetch.VPCConfig = lambda.VpcConfig{}
	s.fetch.Permissions = iam.GetJoinAndFetchLambdaPolicy()
	s.fetch.Init()

	s.scale.TableName = s.TableName
	s.scale.ASGName = s.ASGName
	s.scale.HostGroupInfo = s.HostGroupInfo
	s.scale.Type = lambdas.LambdaScale
	s.scale.VPCConfig = vpcConfig
	s.scale.Permissions = iam.GetScaleLambdaPolicy()
	s.scale.Init()

	s.terminate.TableName = s.TableName
	s.terminate.ASGName = s.ASGName
	s.terminate.HostGroupInfo = s.HostGroupInfo
	s.terminate.Type = lambdas.LambdaTerminate
	s.terminate.VPCConfig = lambda.VpcConfig{}
	s.terminate.Permissions = iam.GetTerminateLambdaPolicy()
	s.terminate.Init()

	s.transient.TableName = s.TableName
	s.transient.ASGName = s.ASGName
	s.transient.HostGroupInfo = s.HostGroupInfo
	s.transient.Type = lambdas.LambdaTransient
	s.transient.VPCConfig = lambda.VpcConfig{}
	s.transient.Permissions = iam.PolicyDocument{}
	s.transient.Init()
}
