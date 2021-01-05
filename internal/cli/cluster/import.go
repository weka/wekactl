package cluster

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var importParams struct {
	name     string
	username string
	password string
}

var importCmd = &cobra.Command{
	Use:   "import [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			err := cluster.ImportCluster(importParams.name, importParams.username, importParams.password)
			if err != nil {
				logging.UserFailure("Import failed!")
				log.Debug().Msg(err.Error())
				return err
			}
			logging.UserSuccess("Import finished successfully!")
		} else {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			logging.UserFailure(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	importCmd.Flags().StringVarP(&importParams.name, "name", "n", "", "EKS cluster name")
	importCmd.Flags().StringVarP(&importParams.username, "username", "u", "", "Cluster username")
	importCmd.Flags().StringVarP(&importParams.password, "password", "p", "", "Cluster password")
	_ = importCmd.MarkFlagRequired("name")
	_ = importCmd.MarkFlagRequired("username")
	_ = importCmd.MarkFlagRequired("password")
}
