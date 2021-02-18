package cluster

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/dist"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/aws/scalemachine"
	"wekactl/internal/cluster"
)

type ScaleMachine struct {
	Arn             string
	TableName       string
	Version         string
	HostGroupInfo   hostgroups.HostGroupInfo
	HostGroupParams hostgroups.HostGroupParams
	fetch           Lambda
	scale           Lambda
	terminate       Lambda
	transient       Lambda
	StateMachine    scalemachine.StateMachine
	Profile         IamProfile
}

func (s *ScaleMachine) SubResources() []cluster.Resource {
	return []cluster.Resource{&s.fetch, &s.scale, &s.terminate, &s.transient, &s.Profile}
}

func (s *ScaleMachine) ResourceName() string {
	return common.GenerateResourceName(s.HostGroupInfo.ClusterName, s.HostGroupInfo.Name)
}

func (s *ScaleMachine) Fetch() error {
	version, err := db.GetResourceVersion(s.TableName, "scalemachine", "", s.HostGroupInfo.Name)
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
	return dist.LambdasID + s.Profile.TargetVersion()
}

func (s *ScaleMachine) Delete() error {
	err := s.Profile.Delete()
	if err != nil {
		return err
	}

	err = s.fetch.Delete()
	if err != nil {
		return err
	}

	err = s.scale.Delete()
	if err != nil {
		return err
	}

	err = s.terminate.Delete()
	if err != nil {
		return err
	}

	err = s.transient.Delete()
	if err != nil {
		return err
	}

	return scalemachine.DeleteStateMachine(s.ResourceName())
}

func (s *ScaleMachine) Create() (err error) {
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
	return db.SaveResourceVersion(s.TableName, "scalemachine", "", s.HostGroupInfo.Name, s.TargetVersion())
}

func (s *ScaleMachine) Update() error {
	panic("implement me")
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
	s.fetch.HostGroupInfo = s.HostGroupInfo
	s.fetch.Type = lambdas.LambdaFetchInfo
	s.fetch.VPCConfig = lambda.VpcConfig{}
	s.fetch.Permissions = iam.GetJoinAndFetchLambdaPolicy()
	s.fetch.Init()

	s.scale.TableName = s.TableName
	s.scale.HostGroupInfo = s.HostGroupInfo
	s.scale.Type = lambdas.LambdaScale
	s.scale.VPCConfig = vpcConfig
	s.scale.Permissions = iam.GetScaleLambdaPolicy()
	s.scale.Init()

	s.terminate.TableName = s.TableName
	s.terminate.HostGroupInfo = s.HostGroupInfo
	s.terminate.Type = lambdas.LambdaTerminate
	s.terminate.VPCConfig = lambda.VpcConfig{}
	s.terminate.Permissions = iam.GetTerminateLambdaPolicy()
	s.terminate.Init()

	s.transient.TableName = s.TableName
	s.transient.HostGroupInfo = s.HostGroupInfo
	s.transient.Type = lambdas.LambdaTransient
	s.transient.VPCConfig = lambda.VpcConfig{}
	s.transient.Permissions = iam.PolicyDocument{}
	s.transient.Init()
}
