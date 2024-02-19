package cluster

import (
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/kms"
	"wekactl/internal/cluster"
)

const dbVersion = "v2"

type DynamoDb struct {
	ClusterName cluster.ClusterName
	StackId     string
	Version     string
	KmsKey      KmsKey
	//ClusterSettings db.ClusterSettings
}

func (d *DynamoDb) Tags() cluster.Tags {
	return cluster.GetCommonResourceTags(d.ClusterName, d.TargetVersion()).Update(
		cluster.Tags{"wekactl.io/stack_id": d.StackId})
}

func (d *DynamoDb) SubResources() []cluster.Resource {
	return []cluster.Resource{&d.KmsKey}
}

func (d *DynamoDb) ResourceName() string {
	return db.GetTableName(d.ClusterName)
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

func (d *DynamoDb) Create(tags cluster.Tags) error {
	return db.CreateDb(d.ResourceName(), d.KmsKey.Key, tags)
}

func (d *DynamoDb) Update(tags cluster.Tags) error {
	err := db.UpdateDbVersion(d.ClusterName, tags)
	if err != nil {
		return err
	}

	var creds db.ClusterCreds
	if d.DeployedVersion() == "v1" {
		creds, err = db.GetUsernameAndPassword(d.ResourceName())
		if err != nil {
			return err
		}
		err = db.ChangeCredentials(d.ResourceName(), creds.Username, creds.Password)
	}

	return err
}
