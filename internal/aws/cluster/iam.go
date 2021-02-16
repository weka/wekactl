package cluster

import (
	"fmt"
	"github.com/google/uuid"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	strings2 "wekactl/internal/lib/strings"
)

type IamProfile struct {
	Arn              string
	Name             string
	PolicyName       string
	TableName        string
	Version          string
	AssumeRolePolicy iam.AssumeRolePolicyDocument
	HostGroupInfo    hostgroups.HostGroupInfo
	Policy           iam.PolicyDocument
}

func (i *IamProfile) resourceNameBase() string {
	name := common.GenerateResourceName(i.HostGroupInfo.ClusterName, i.HostGroupInfo.Name)
	return fmt.Sprintf("%s-%s", name, i.Name)
}

func (i *IamProfile) ResourceName() string {
	//creating and deleting the same role name and use it for lambda caused problems, so we use unique uuid
	return strings2.ElfHashSuffixed(fmt.Sprintf("%s-%s", i.resourceNameBase(), uuid.New().String()), 64)
}

func (i *IamProfile) Fetch() error {
	version, err := db.GetResourceVersion(i.TableName, "iam", i.Name, i.HostGroupInfo.Name)
	if err != nil {
		return err
	}

	i.Version = version
	return nil
}

func (i *IamProfile) Init() {
	return
}

func (i *IamProfile) DeployedVersion() string {
	return i.Version
}

func (i *IamProfile) TargetVersion() string {
	return i.Policy.VersionHash()
}

func (i *IamProfile) Delete() error {
	return iam.DeleteIamRole(i.resourceNameBase(), i.PolicyName)
}

func (i *IamProfile) Create() error {
	arn, err := iam.CreateIamRole(i.HostGroupInfo, i.ResourceName(), i.PolicyName, i.AssumeRolePolicy, i.Policy)
	if err != nil {
		return err
	}

	i.Arn = *arn

	return db.SaveResourceVersion(i.TableName, "iam", i.Name, i.HostGroupInfo.Name, i.TargetVersion())
}

func (i *IamProfile) Update() error {
	panic("implement me")
}
