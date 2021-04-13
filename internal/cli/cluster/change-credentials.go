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

var changeCredsParams struct {
	Name     string
	Username string
	Password string
}

var changeCredentialsCmd = &cobra.Command{
	Use:   "change-credentials [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			tableNAme := common.GenerateResourceName(cluster.ClusterName(changeCredsParams.Name), "")
			err := db.ChangeCredentials(tableNAme, changeCredsParams.Username, changeCredsParams.Password)
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
	changeCredentialsCmd.Flags().StringVarP(&changeCredsParams.Name, "name", "n", "", "weka cluster name")
	changeCredentialsCmd.Flags().StringVarP(&changeCredsParams.Username, "username", "u", "", "Cluster username")
	changeCredentialsCmd.Flags().StringVarP(&changeCredsParams.Password, "password", "p", "", "Cluster password")
	_ = changeCredentialsCmd.MarkFlagRequired("name")
	_ = changeCredentialsCmd.MarkFlagRequired("username")
	_ = changeCredentialsCmd.MarkFlagRequired("password")
}
