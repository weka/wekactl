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

var cleanCmd = &cobra.Command{
	Use:   "clean [flags]",
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

			if keepInstances {
				autoscaling.KeepInstances = true
			}

			awsCluster.Init()
			err := cluster.CleanResource(&awsCluster)

			if err != nil {
				logging.UserFailure("Cleaning failed!")
				return err
			}
			logging.UserSuccess("Cleaning finished successfully!")
		} else {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			logging.UserFailure(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	cleanCmd.Flags().StringVarP(&StackName, "name", "n", "", "EKS cluster name")
	cleanCmd.Flags().BoolVarP(&keepInstances, "keep-instances", "k", false, "Keep instances")
	_ = cleanCmd.MarkFlagRequired("name")
}
