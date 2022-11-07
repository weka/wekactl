package cluster

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/rs/zerolog/log"
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

func migrateSettings(clusterName cluster.ClusterName, dbClusterSettings *db.ClusterSettings, hostGroups []HostGroup) error {
	// this function is for updating old clusters that might be missing some settings in db or clusters with wrong volumes info

	log.Debug().Msg("Checking if settings migration is needed ...")
	migrateRequired := false

	if dbClusterSettings.VpcId == "" {
		vpcId, err := common.VpcBySubnet(dbClusterSettings.Subnet)
		if err != nil {
			return err
		}
		dbClusterSettings.VpcId = vpcId
		migrateRequired = true
	}

	versionInfo, err := env.GetBuildVersion()
	if err != nil {
		return err
	}
	if dbClusterSettings.BuildVersion != versionInfo.BuildVersion {
		dbClusterSettings.BuildVersion = versionInfo.BuildVersion
		migrateRequired = true
	}

	for _, hostGroup := range hostGroups {
		if hostGroup.HostGroupInfo.Role == common.RoleBackend {
			if dbClusterSettings.Backends.VolumesInfo == nil {
				dbClusterSettings.Backends.VolumesInfo = hostGroup.HostGroupParams.VolumesInfo
				migrateRequired = true
			}
		}
		if hostGroup.HostGroupInfo.Role == common.RoleClient {
			if dbClusterSettings.Clients.VolumesInfo == nil {
				dbClusterSettings.Clients.VolumesInfo = hostGroup.HostGroupParams.VolumesInfo
				migrateRequired = true
			}
		}
	}

	if !migrateRequired {
		return nil
	}

	log.Debug().Msg("Settings migration is needed, saving to DB ...")
	if err := db.SaveClusterSettings(db.GetTableName(clusterName), *dbClusterSettings); err != nil {
		return err
	}

	return nil
}

func GetCluster(name cluster.ClusterName, fetchHotGroupParams bool) (awsCluster AWSCluster, err error) {
	dbClusterSettings, err := db.GetClusterSettings(name)
	if err != nil {
		if _, ok := err.(*dynamodb.ResourceNotFoundException); ok {
			err = errors.New(fmt.Sprintf("Cluster doesn't exist in %s", env.Config.Region))
		}
		return
	}

	hostGroups, err := getHostGroups(name, fetchHotGroupParams)
	if err != nil {
		return
	}

	err = migrateSettings(name, &dbClusterSettings, hostGroups)
	if err != nil {
		return
	}

	awsCluster = AWSCluster{
		Name:            name,
		ClusterSettings: dbClusterSettings,
		HostGroups:      hostGroups,
	}
	awsCluster.Init()
	return

}
