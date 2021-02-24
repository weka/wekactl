package cluster

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
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
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	fetch           Lambda
	scale           Lambda
	terminate       Lambda
	transient       Lambda
	StateMachine    scalemachine.StateMachine
	Profile         IamProfile
}

func (s *ScaleMachine) Tags() interface{} {
	return scalemachine.GetStateMachineTags(s.HostGroupInfo, s.TargetVersion())
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
	return nil
}

func (s *ScaleMachine) DeployedVersion() string {
	return s.Version
}

func (s *ScaleMachine) TargetVersion() string {
	return scaleMachineVersion
}

func (s *ScaleMachine) Delete() error {
	return scalemachine.DeleteStateMachine(s.ResourceName())
}

func (s *ScaleMachine) Create() (err error) {
	stateMachineLambdasArn := scalemachine.StateMachineLambdasArn{
		Fetch:     s.fetch.Arn,
		Scale:     s.scale.Arn,
		Terminate: s.terminate.Arn,
		Transient: s.transient.Arn,
	}

	arn, err := scalemachine.CreateStateMachine(s.Tags().([]*sfn.Tag), stateMachineLambdasArn, s.Profile.Arn, s.ResourceName())
	if err != nil {
		return
	}
	s.Arn = *arn
	return nil
}

func (s *ScaleMachine) Update() error {
	err := s.Delete()
	if err != nil {
		return err
	}
	return s.Create()
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
