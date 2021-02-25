package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	cluster2 "wekactl/internal/aws/cluster"
	"wekactl/internal/aws/db"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var updateCmd = &cobra.Command{
	Use:   "update [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {

			clusterName := cluster.ClusterName(StackName)

			backendsHostGroup, err := cluster2.GenerateHostGroupFromLaunchTemplate(
				clusterName, hostgroups.RoleBackend, "Backends")
			if err != nil {
				return err
			}

			clientsHostGroup, err := cluster2.GenerateHostGroupFromLaunchTemplate(
				clusterName, hostgroups.RoleClient, "Clients")
			if err != nil {
				return err
			}

			dynamoDb := cluster2.DynamoDb{
				ClusterName: clusterName,
			}

			awsCluster := cluster2.AWSCluster{
				Name:          clusterName,
				DefaultParams: db.DefaultClusterParams{},
				CFStack: cluster2.Stack{
					StackName: StackName,
				},
				DynamoDb: dynamoDb,
				HostGroups: []cluster2.HostGroup{
					backendsHostGroup,
					clientsHostGroup,
				},
			}

			awsCluster.Init()
			err = cluster.EnsureResource(&awsCluster)

			if err != nil {
				logging.UserFailure("Update failed!")
				return err
			}
			logging.UserSuccess("Update finished successfully!")
		} else {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			logging.UserFailure(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	updateCmd.Flags().StringVarP(&StackName, "name", "n", "", "EKS cluster name")
	_ = updateCmd.MarkFlagRequired("name")
}
