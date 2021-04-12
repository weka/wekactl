package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/kms"
	"wekactl/internal/cluster"
)

const dbVersion = "v1"

type DynamoDb struct {
	ClusterName cluster.ClusterName
	Username    string
	Password    string
	StackId     string
	Version     string
	KmsKey      KmsKey
	ClusterSettings cluster.ClusterSettings
}

func (d *DynamoDb) Tags() cluster.Tags {
	return cluster.GetCommonResourceTags(d.ClusterName, d.TargetVersion()).Update(
		cluster.Tags{"wekactl.io/stack_id": d.StackId})
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

	if d.KmsKey.Key == "" {
		kmsKeyId, err := kms.GetKMSKeyId(d.ClusterName)
		if err != nil {
			return err
		}
		d.KmsKey.Key = kmsKeyId
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
	return db.DeleteDB(d.ResourceName())
}

func (d *DynamoDb) Create(tags cluster.Tags) error {
	err := db.CreateDb(d.ResourceName(), d.KmsKey.Key, tags)
	if err != nil {
		return err
	}
	err = db.SaveCredentials(d.ResourceName(), d.Username, d.Password)
	if err != nil {
		return err
	}
	return db.SaveClusterSettings(d.ResourceName(), d.ClusterSettings)
}

func (d *DynamoDb) Update() error {
	panic("update not supported")
}
