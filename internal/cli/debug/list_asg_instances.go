package debug

import (
	"github.com/spf13/cobra"
	"gopkg.in/errgo.v2/fmt/errors"
	"wekactl/internal/aws/debug"
	"wekactl/internal/env"
)


var listAsgInstances = &cobra.Command{
	Use:   "list-asg-instances",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			return debug.RenderASGInstancesTable(ASGName)
		} else {
			return errors.Newf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
	},
}

func init() {
	listAsgInstances.Flags().StringVarP(&ASGName, "name", "n", "", "AWS Autoscaling group name")
	listAsgInstances.MarkFlagRequired("name")
	Debug.AddCommand(listAsgInstances)
}
