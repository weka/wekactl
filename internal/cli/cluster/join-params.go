package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/alb"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var joinParamsCmd = &cobra.Command{
	Use:   "join-params [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {

			err := alb.PrintStatelessClientsJoinScript(cluster.ClusterName(StackName), importParams.DnsAlias)
			if err != nil {
				return err
			}

		} else {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			logging.UserFailure(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	joinParamsCmd.Flags().StringVarP(&StackName, "name", "n", "", "weka cluster name")
	joinParamsCmd.Flags().StringVarP(&importParams.DnsAlias, "dns-alias", "l", "", "ALB dns alias")
	_ = joinParamsCmd.MarkFlagRequired("name")
}
