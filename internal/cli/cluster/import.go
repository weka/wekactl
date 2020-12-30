package cluster

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	"wekactl/internal/logging"
)

var importCmd = &cobra.Command{
	Use:   "import [flags]",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if Provider == "aws" {
			err := cluster.ImportCluster(Region, Name, Username, Password)
			if err != nil {
				logging.UserFailure("Import failed!")
				log.Debug().Msg(err.Error())
			} else {
				logging.UserSuccess("Import finished successfully!")
			}
		} else {
			logging.UserFailure("Cloud provider '%s' is not supported with this action", Provider)
		}
	},
}

func init() {
	importCmd.Flags().StringVarP(&Name, "name", "n", "", "EKS cluster name")
	importCmd.Flags().StringVarP(&Username, "username", "u", "", "Cluster username")
	importCmd.Flags().StringVarP(&Password, "password", "p", "", "Cluster password")
	importCmd.MarkFlagRequired("name")
	importCmd.MarkFlagRequired("username")
	importCmd.MarkFlagRequired("password")
	Cluster.AddCommand(importCmd)
}
