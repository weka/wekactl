package cleaner

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type DynamoDb struct {
	Table       *dynamodb.TableDescription
	ClusterName cluster.ClusterName
}

func (d *DynamoDb) Fetch(clusterName cluster.ClusterName) error {
	table, err := db.GetClusterDb(clusterName)
	if err != nil {
		return err
	}
	d.Table = table
	d.ClusterName = clusterName
	return nil
}

func (d *DynamoDb) Delete() error {
	return db.DeleteTable(d.Table, d.ClusterName)
}

func (d *DynamoDb) Print() {
	logging.UserInfo("DynamoDb:")
	if d.Table != nil {
		logging.UserInfo("\t- %s", *d.Table.TableName)
	}
}
