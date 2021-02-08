package cluster

import (
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
)

type IamProfile struct {
	Arn              string
	Name             string
	PolicyName       string
	AssumeRolePolicy iam.AssumeRolePolicyDocument
	HostGroupInfo    hostgroups.HostGroupInfo
	Policy           iam.PolicyDocument
}

func (i *IamProfile) ResourceName() string {
	return i.Name
}

func (i *IamProfile) Fetch() error {
	return nil
}

func (i *IamProfile) Init() {
	return
}

func (i *IamProfile) DeployedVersion() string {
	return ""
}

func (i *IamProfile) TargetVersion() string {
	return i.Policy.VersionHash()
}

func (i *IamProfile) Delete() error {
	panic("implement me")
}

func (i *IamProfile) Create() error {
	arn, err := iam.CreateIamRole(i.HostGroupInfo, i.Name, i.PolicyName, i.AssumeRolePolicy, i.Policy)
	if err != nil {
		return err
	}

	i.Arn = *arn
	return nil
}

func (i *IamProfile) Update() error {
	panic("implement me")
}
