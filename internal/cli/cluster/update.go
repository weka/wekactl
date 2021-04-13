package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var updateCmd = &cobra.Command{
	Use:   "update [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			err := cluster.UpdateCluster(StackName)
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
	updateCmd.Flags().StringVarP(&StackName, "name", "n", "", "weka cluster name")
	_ = updateCmd.MarkFlagRequired("name")
}
