package cluster

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cleaner"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var dryRun bool

var cleanupCmd = &cobra.Command{
	Use:   "cleanup [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if env.Config.Provider == "aws" {

			clusterName := cluster.ClusterName(StackName)

			if dryRun {
				logging.UserInfo("This is dry run, running cleanup will remove the following resources:")
			} else {
				logging.UserInfo("Removing the following resources:")
			}

			err = cluster.CleanupResource(&cleaner.IamProfile{}, clusterName, dryRun)
			if err != nil {
				return
			}

			err = cluster.CleanupResource(&cleaner.Lambda{}, clusterName, dryRun)
			if err != nil {
				return
			}

			err = cluster.CleanupResource(&cleaner.ApiGateway{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			err = cluster.CleanupResource(&cleaner.LaunchTemplate{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			err = cluster.CleanupResource(&cleaner.ScaleMachine{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			err = cluster.CleanupResource(&cleaner.CloudWatch{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			err = cluster.CleanupResource(&cleaner.AutoscalingGroup{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			err = cluster.CleanupResource(&cleaner.ApplicationLoadBalancer{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			err = cluster.CleanupResource(&cleaner.KmsKey{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			err = cluster.CleanupResource(&cleaner.DynamoDb{}, clusterName, dryRun)
			if err != nil {
				return err
			}

			ids, err := common.GetClusterInstances(clusterName)
			if err != nil {
				return err
			}
			logging.UserInfo("InstanceIds:")
			for _, id := range ids {
				logging.UserInfo("\t- %s", id)
			}

			if !dryRun {
				err = common.DeleteClusterInstanceIds(ids)
				if err != nil {
					log.Error().Err(err)
				}
			}

			logging.UserSuccess("Cleanup finished successfully!")
		} else {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			logging.UserFailure(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	cleanupCmd.Flags().StringVarP(&StackName, "name", "n", "", "weka cluster name")
	cleanupCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "dry run")
	_ = cleanupCmd.MarkFlagRequired("name")
}
