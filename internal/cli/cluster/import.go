package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cluster"
	cluster2 "wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var importParams cluster2.ImportParams

var importCmd = &cobra.Command{
	Use:   "import [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			err := cluster.ImportCluster(importParams)
			if err != nil {
				logging.UserFailure("Import failed!")
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
	importCmd.Flags().StringVarP(&importParams.Name, "name", "n", "", "weka cluster name")
	importCmd.Flags().StringVarP(&importParams.Username, "username", "u", "", "cluster admin username")
	importCmd.Flags().StringVarP(&importParams.Password, "password", "p", "", "cluster admin password")
	importCmd.Flags().StringArrayVarP(&importParams.TagsList, "tags", "t", []string{}, "cloud resources tags, each tag should be passed in this pattern: '-t key=value'")
	importCmd.Flags().BoolVarP(&importParams.PrivateSubnet, "private-subnet", "s", false, "cluster runs in private subnet, requires execute-api VPC endpoint to present on VPC")
	importCmd.Flags().StringVarP(&importParams.AdditionalAlbSubnet, "additional-alb-subnet", "a", "", "Additional subnet to use for ALB")
	_ = importCmd.MarkFlagRequired("name")
	_ = importCmd.MarkFlagRequired("username")
	_ = importCmd.MarkFlagRequired("password")
}
