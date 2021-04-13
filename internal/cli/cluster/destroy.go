package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/autoscaling"
	cluster2 "wekactl/internal/aws/cluster"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var keepInstances bool

var destroyCmd = &cobra.Command{
	Use:   "destroy [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {

			clusterName := cluster.ClusterName(StackName)

			backendsHostGroup := cluster2.GenerateHostGroup(
				clusterName,
				common.HostGroupParams{},
				common.RoleBackend,
				"Backends",
			)

			clientsHostGroup := cluster2.GenerateHostGroup(
				clusterName,
				common.HostGroupParams{},
				common.RoleClient,
				"Clients",
			)

			dynamoDb := cluster2.DynamoDb{
				ClusterName: clusterName,
			}
			dynamoDb.Init()
			err := cluster.DestroyResource(&dynamoDb)
			if err != nil {
				return err
			}

			awsCluster := cluster2.AWSCluster{
				Name:          clusterName,
				DefaultParams: db.ClusterSettings{},
				CFStack: cluster2.Stack{
					StackName: StackName,
				},
				HostGroups: []cluster2.HostGroup{
					backendsHostGroup,
					clientsHostGroup,
				},
			}

			if keepInstances {
				autoscaling.KeepInstances = true
			}

			awsCluster.Init()
			err = cluster.DestroyResource(&awsCluster)
			if err != nil {
				logging.UserFailure("Destroying failed!")
				return err
			}
			logging.UserSuccess("Destroying finished successfully!")
		} else {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			logging.UserFailure(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	destroyCmd.Flags().StringVarP(&StackName, "name", "n", "", "EKS cluster name")
	destroyCmd.Flags().BoolVarP(&keepInstances, "keep-instances", "k", false, "Keep instances")
	_ = destroyCmd.MarkFlagRequired("name")
}
