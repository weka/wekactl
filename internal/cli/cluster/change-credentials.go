package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var changeCredentialsCmd = &cobra.Command{
	Use:   "change-credentials [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			tableNAme := common.GenerateResourceName(cluster.ClusterName(importParams.name), "")
			err := db.ChangeCredentials(tableNAme, importParams.username, importParams.password)
			if err != nil {
				logging.UserFailure("Credentials change failed!")
				return err
			}
			logging.UserSuccess("Credentials change finished successfully!")
		} else {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			logging.UserFailure(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	changeCredentialsCmd.Flags().StringVarP(&importParams.name, "name", "n", "", "EKS cluster name")
	changeCredentialsCmd.Flags().StringVarP(&importParams.username, "username", "u", "", "Cluster username")
	changeCredentialsCmd.Flags().StringVarP(&importParams.password, "password", "p", "", "Cluster password")
	_ = changeCredentialsCmd.MarkFlagRequired("name")
	_ = changeCredentialsCmd.MarkFlagRequired("username")
	_ = changeCredentialsCmd.MarkFlagRequired("password")
}
