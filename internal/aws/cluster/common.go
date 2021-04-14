package cluster

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
)

func GenerateHostGroup(clusterName cluster.ClusterName, hostGroupParams common.HostGroupParams, role common.InstanceRole, name common.HostGroupName) HostGroup {
	hostGroupInfo := common.HostGroupInfo{
		Name:        name,
		Role:        role,
		ClusterName: clusterName,
	}

	hostGroup := HostGroup{
		HostGroupInfo:   hostGroupInfo,
		HostGroupParams: hostGroupParams,
	}

	return hostGroup
}

func GetCluster(name cluster.ClusterName) (awsCluster AWSCluster, err error) {
	dbClusterSettings, err := db.GetClusterSettings(name)
	if err != nil {
		if _, ok := err.(*dynamodb.ResourceNotFoundException); ok {
			err = errors.New(fmt.Sprintf("Cluster doesn't exist in %s", env.Config.Region))
		}
		return
	}

	backendsHostGroup, err := generateHostGroupFromLaunchTemplate(
		name, common.RoleBackend, "Backends")
	if err != nil {
		return
	}

	clientsHostGroup, err := generateHostGroupFromLaunchTemplate(
		name, common.RoleClient, "Clients")
	if err != nil {
		return
	}

	hostGroups := append(backendsHostGroup, clientsHostGroup...)

	awsCluster = AWSCluster{
		Name:            name,
		ClusterSettings: dbClusterSettings,
		HostGroups:      hostGroups,
	}
	awsCluster.Init()
	return

}
