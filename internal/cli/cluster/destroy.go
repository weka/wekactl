package cluster

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/aws/cleaner"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var keepInstances, dryRun bool

var destroyCmd = &cobra.Command{
	Use:   "destroy [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {

			clusterName := cluster.ClusterName(StackName)

			if keepInstances {
				// TODO: Evicting instances manually and then running destroy would be better, without hacks
				autoscaling.KeepInstances = true
			}

			if dryRun {
				logging.UserInfo("This is dry run, running cleanup will remove the following resources:")
			} else {
				logging.UserInfo("Removing the following resources:")
			}

			resources := []cluster.Cleaner{
				&cleaner.IamProfile{ClusterName: clusterName},
				&cleaner.Lambda{ClusterName: clusterName},
				&cleaner.ApiGateway{ClusterName: clusterName},
				&cleaner.LaunchTemplate{ClusterName: clusterName},
				&cleaner.ScaleMachine{ClusterName: clusterName},
				&cleaner.CloudWatch{ClusterName: clusterName},
				&cleaner.AutoscalingGroup{ClusterName: clusterName},
				&cleaner.ApplicationLoadBalancer{ClusterName: clusterName},
				&cleaner.DynamoDb{ClusterName: clusterName},
				&cleaner.KmsKey{ClusterName: clusterName},
			}

			for _, r := range resources {
				if err := cluster.CleanupResource(r, dryRun); err != nil {
					return err
				}

			}

			if !keepInstances {
				ids, err := common.GetClusterInstances(clusterName)
				if err != nil {
					return err
				}
				logging.UserInfo("InstanceIds:")
				for _, id := range ids {
					logging.UserInfo("\t- %s", id)
				}
				if !dryRun {
					err = common.DeleteInstances(ids)
					if err != nil {
						log.Error().Err(err)
					}
				}
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
	destroyCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "dry run")
	_ = destroyCmd.MarkFlagRequired("name")
}
