package cluster

import (
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
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

func GetCluster(name cluster.ClusterName) (awsCluster AWSCluster, err error){
	dbClusterSettings, err := db.GetClusterSettings(name)
	if err != nil {
		return
	}

	//TODO: This might not be single one
	backendsHostGroup, err := generateHostGroupFromLaunchTemplate(
		name, common.RoleBackend, "Backends")
	if err != nil {
		return
	}

	//TODO: This might not be single one
	clientsHostGroup, err := generateHostGroupFromLaunchTemplate(
		name, common.RoleClient, "Clients")
	if err != nil {
		return
	}

	awsCluster = AWSCluster{
		Name:            name,
		ClusterSettings: dbClusterSettings,
		HostGroups: []HostGroup{
			backendsHostGroup,
			clientsHostGroup,
		},
	}
	awsCluster.Init()
	return

}