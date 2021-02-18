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

func (d *DynamoDb) SubResources() []cluster.Resource {
	return []cluster.Resource{&d.KmsKey}
}

func (d *DynamoDb) ResourceName() string {
	return common.GenerateResourceName(d.ClusterName, "")
}

func (d *DynamoDb) Fetch() error {
	exists, err := db.Exists(d.ResourceName())
	if err != nil {
		return err
	}
	if exists {
		d.Version = dbVersion
	}
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
	err := d.KmsKey.Delete()
	if err != nil {
		return err
	}
	return db.DeleteDB(d.ResourceName())
}

func (d *DynamoDb) Create() error {
	err := db.CreateDb(d.ResourceName(), d.KmsKey.Key, common.GetCommonTags(d.ClusterName).Update(common.Tags{
		"wekactl.io/stack_id": d.StackId}))
	if err != nil {
		return err
	}

	err = db.SaveCredentials(d.ResourceName(), d.Username, d.Password)
	if err != nil {
		return err
	}
	return db.SaveResourceVersion(d.ResourceName(), "kms", "", "", d.KmsKey.TargetVersion())
}

func (d *DynamoDb) Update() error {
	panic("implement me")
}
