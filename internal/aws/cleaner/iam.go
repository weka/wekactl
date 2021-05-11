package cleaner

import (
	"github.com/aws/aws-sdk-go/service/iam"
	iam2 "wekactl/internal/aws/iam"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type IamProfile struct {
	ClusterName cluster.ClusterName
	Roles       []*iam.Role
	Policies    map[string][]*iam.AttachedPolicy
}

func (i *IamProfile) Fetch() error {
	roles, err := iam2.GetClusterRoles(i.ClusterName)
	if err != nil {
		return err
	}
	i.Roles = roles

	policies, err := iam2.GetRolesPolicies(roles)
	if err != nil {
		return err
	}
	i.Policies = policies

	return nil
}

func (i *IamProfile) Delete() error {
	return iam2.DeleteRolesAndPolicies(i.Roles, i.Policies)
}

func (i *IamProfile) Print() {
	logging.UserInfo("Roles:")
	for role, policies := range i.Policies {
		logging.UserInfo("\t- %s", role)
		for _, policy := range policies {
			logging.UserInfo("\t\t- policy:%s", *policy.PolicyName)
		}
	}
}
