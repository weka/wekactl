package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/autoscaling"
	cluster2 "wekactl/internal/aws/cluster"
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

			awsCluster, err := cluster2.GetCluster(clusterName, false)
			if err != nil {
				return err
			}

			if keepInstances {
				// TODO: Evicting instances manually and then running destroy would be better, without hacks
				autoscaling.KeepInstances = true
			}

			err = cluster.DestroyResource(&awsCluster)
			if err != nil {
				logging.UserFailure("Destroying failed!")
				return err
			}

			dynamoDb := cluster2.DynamoDb{
				ClusterName: clusterName,
			}
			dynamoDb.Init()
			err = cluster.DestroyResource(&dynamoDb)
			if err != nil {
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
	destroyCmd.Flags().StringVarP(&StackName, "name", "n", "", "weka cluster name")
	destroyCmd.Flags().BoolVarP(&keepInstances, "keep-instances", "k", false, "Keep instances")
	destroyCmd.Flags().MarkHidden("keep-instances")
	_ = destroyCmd.MarkFlagRequired("name")
}
