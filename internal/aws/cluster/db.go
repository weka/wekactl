package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
)

/*
type DynamoDb struct{
	KMSKey KmsKey
}
*/

const dbVersion = "v1"

type DynamoDb struct {
	ClusterName cluster.ClusterName
	Username    string
	Password    string
	StackId     string
	Version     string
	KmsKey      KmsKey
}

func (d *DynamoDb) Tags() interface{} {
	return common.GetCommonTags(d.ClusterName, d.TargetVersion()).Update(
		common.Tags{"wekactl.io/stack_id": d.StackId})
}

func (d *DynamoDb) SubResources() []cluster.Resource {
	return []cluster.Resource{&d.KmsKey}
}

func (d *DynamoDb) ResourceName() string {
	return common.GenerateResourceName(d.ClusterName, "")
}

func (d *DynamoDb) Fetch() error {
	version, err := db.GetDbVersion(d.ResourceName())
	if err != nil {
		return err
	}
	d.Version = version
	return nil
}

func (d *DynamoDb) Init() {
	log.Debug().Msgf("Initializing db ...")
	d.KmsKey.ClusterName = d.ClusterName
}

func (d *DynamoDb) DeployedVersion() string {
	return d.Version
}

func (d *DynamoDb) TargetVersion() string {
	return dbVersion
}

func (d *DynamoDb) Delete() error {
	return db.DeleteDB(d.ResourceName())
}

func (d *DynamoDb) Create() error {
	err := db.CreateDb(d.ResourceName(), d.KmsKey.Key, d.Tags().(common.Tags))
	if err != nil {
		return err
	}
	return db.SaveCredentials(d.ResourceName(), d.Username, d.Password)
}

func (d *DynamoDb) Update() error {
	panic("implement me")
}
