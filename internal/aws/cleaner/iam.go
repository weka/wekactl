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
}

func (i *IamProfile) Fetch() error {
	roles, err := iam2.GetClusterRoles(i.ClusterName)
	if err != nil {
		return err
	}
	i.Roles = roles

	return nil
}

func (i *IamProfile) Delete() error {
	return iam2.DeleteRoles(i.ClusterName, i.Roles)
}

func (i *IamProfile) Print() {
	logging.UserInfo("Roles:")
	for _, role := range i.Roles {
		logging.UserInfo("\t- %s", *role.RoleName)
	}
}
