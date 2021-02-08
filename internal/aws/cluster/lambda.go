package cluster

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/dist"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/cluster"
	strings2 "wekactl/internal/lib/strings"
)

type Lambda struct {
	Arn           string
	Type          lambdas.LambdaType
	Profile       IamProfile
	VPCConfig     lambda.VpcConfig
	HostGroupInfo hostgroups.HostGroupInfo
	Permissions   iam.PolicyDocument
}

func (l *Lambda) ResourceName() string {
	n := strings.Join([]string{"wekactl", string(l.HostGroupInfo.ClusterName), string(l.Type), string(l.HostGroupInfo.Name)}, "-")
	return strings2.ElfHashSuffixed(n, 64)
}

func (l *Lambda) Fetch() error {
	//searchTags := GetHostGroupTags(l.HgInfo).Update()
	//panic("implement me")
	return nil
}

func (l *Lambda) Init() {
	log.Debug().Msgf("Initializing hostgroup %s %s lambda ...", string(l.HostGroupInfo.Name), string(l.Type))
	//creating and deleting the same role name and use it for lambda caused problems, so we use unique uuid
	l.Profile.Name = fmt.Sprintf("wekactl-%s-%s-%s", l.HostGroupInfo.Name, string(l.Type), uuid.New().String())
	l.Profile.PolicyName = l.ResourceName()
	l.Profile.AssumeRolePolicy = iam.GetLambdaAssumeRolePolicy()
	l.Profile.HostGroupInfo = l.HostGroupInfo
	l.Profile.Policy = l.Permissions
	l.Profile.Init()
}

func (l *Lambda) DeployedVersion() string {
	return ""
}

func (l *Lambda) TargetVersion() string {
	return dist.LambdasID + l.Profile.TargetVersion()
}

func (l *Lambda) Delete() error {
	panic("implement me")
}

func (l *Lambda) Create() (err error) {
	err = cluster.EnsureResource(&l.Profile)
	if err != nil {
		return
	}

	functionConfiguration, err := lambdas.CreateLambda(l.HostGroupInfo, l.Type, l.ResourceName(), l.Profile.Arn, l.VPCConfig)
	if err != nil {
		return
	}
	l.Arn = *functionConfiguration.FunctionArn
	return
}

func (l *Lambda) Update() error {
	if l.DeployedVersion() == l.TargetVersion() {
		return nil
	}
	err := l.Profile.Update()
	if err != nil {
		return err
	}
	return nil
}
