package cluster

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/aws/scalemachine"
	"wekactl/internal/cluster"
)

type ScaleMachine struct {
	Arn             string
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	fetch           Lambda
	scale           Lambda
	terminate       Lambda
	transient       Lambda
	StateMachine    scalemachine.StateMachine
	Profile IamProfile
}

func (s *ScaleMachine) ResourceName() string {
	return common.GenerateResourceName(s.HostGroupInfo.ClusterName, s.HostGroupInfo.Name)
}

func (s *ScaleMachine) Fetch() error {
	return nil
}

func (s *ScaleMachine) DeployedVersion() string {
	return ""
}

func (s *ScaleMachine) TargetVersion() string {
	return ""
}

func (s *ScaleMachine) Delete() error {
	panic("implement me")
}

func (s *ScaleMachine) Create() (err error) {
	err = cluster.EnsureResource(&s.Profile)
	if err != nil {
		return
	}

	err = cluster.EnsureResource(&s.fetch)
	if err != nil {
		return
	}

	err = cluster.EnsureResource(&s.scale)
	if err != nil {
		return
	}

	err = cluster.EnsureResource(&s.terminate)
	if err != nil {
		return
	}

	err = cluster.EnsureResource(&s.transient)
	if err != nil {
		return
	}

	stateMachineLambdasArn := scalemachine.StateMachineLambdasArn{
		Fetch:     s.fetch.Arn,
		Scale:     s.scale.Arn,
		Terminate: s.terminate.Arn,
		Transient: s.transient.Arn,
	}

	arn, err := scalemachine.CreateStateMachine(s.HostGroupInfo, stateMachineLambdasArn, s.Profile.Arn, s.ResourceName())
	if err != nil {
		return
	}
	s.Arn = *arn
	return
}

func (s *ScaleMachine) Update() error {
	panic("implement me")
}

func (s *ScaleMachine) Init() {
	log.Debug().Msgf("Initializing hostgroup %s state machine ...", string(s.HostGroupInfo.Name))

	//creating and deleting the same role name and use it for lambda caused problems, so we use unique uuid
	s.Profile.Name = fmt.Sprintf("wekactl-%s-sm-%s", s.HostGroupInfo.Name, uuid.New().String())
	s.Profile.PolicyName = fmt.Sprintf("wekactl-%s-sm-%s", string(s.HostGroupInfo.ClusterName), string(s.HostGroupInfo.Name))
	s.Profile.AssumeRolePolicy = iam.GetStateMachineAssumeRolePolicy()
	s.Profile.HostGroupInfo = s.HostGroupInfo
	s.Profile.Policy = iam.GetStateMachineRolePolicy()
	s.Profile.Init()

	vpcConfig := lambdas.GetLambdaVpcConfig(s.HostGroupParams.Subnet, s.HostGroupParams.SecurityGroupsIds)

	s.fetch.HostGroupInfo = s.HostGroupInfo
	s.fetch.Type = lambdas.LambdaFetchInfo
	s.fetch.VPCConfig = lambda.VpcConfig{}
	s.fetch.Permissions = iam.GetJoinAndFetchLambdaPolicy()
	s.fetch.Init()

	s.scale.HostGroupInfo = s.HostGroupInfo
	s.scale.Type = lambdas.LambdaScale
	s.scale.VPCConfig = vpcConfig
	s.scale.Permissions = iam.GetScaleLambdaPolicy()
	s.scale.Init()

	s.terminate.HostGroupInfo = s.HostGroupInfo
	s.terminate.Type = lambdas.LambdaTerminate
	s.terminate.VPCConfig = lambda.VpcConfig{}
	s.terminate.Permissions = iam.GetTerminateLambdaPolicy()
	s.terminate.Init()

	s.transient.HostGroupInfo = s.HostGroupInfo
	s.transient.Type = lambdas.LambdaTransient
	s.transient.VPCConfig = lambda.VpcConfig{}
	s.transient.Permissions = iam.PolicyDocument{}
	s.transient.Init()
}
