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

type DynamoDb struct {
	ClusterName cluster.ClusterName
	Username    string
	Password    string
	StackId     string
	KmsKey      KmsKey
}

func (d *DynamoDb) ResourceName() string {
	return common.GenerateResourceName(d.ClusterName, "")
}

func (d *DynamoDb) Fetch() error {
	return nil
}

func (d *DynamoDb) Init() {
	log.Debug().Msgf("Initializing db ...")
	d.KmsKey.ClusterName = d.ClusterName
}

func (d *DynamoDb) DeployedVersion() string {
	return ""
}

func (d *DynamoDb) TargetVersion() string {
	return ""
}

func (d *DynamoDb) Delete() error {
	panic("implement me")
}

func (d *DynamoDb) Create() error {
	err := cluster.EnsureResource(&d.KmsKey)
	if err != nil {
		return err
	}

	err = db.CreateDb(d.ResourceName(), d.KmsKey.Key, common.GetCommonTags(d.ClusterName).Update(common.Tags{
		"wekactl.io/stack_id": d.StackId}))
	if err != nil {
		return err
	}

	err = db.SaveCredentials(d.ResourceName(), d.Username, d.Password)
	if err != nil {
		return err
	}
	return nil
}

func (d *DynamoDb) Update() error {
	panic("implement me")
}
