package debug

import (
	"github.com/spf13/cobra"
	"gopkg.in/errgo.v2/fmt/errors"
	"wekactl/internal/aws/autoscaling"
	"wekactl/internal/env"
)

var detachAsgInstances = &cobra.Command{
	Use:   "detach-asg-instances",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			return detach(cmd, args)
		} else {
			return errors.Newf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
	},
}

func detach(cmd *cobra.Command, args []string) error {
	return autoscaling.DetachInstancesFromASG(args, ASGName)
}

func init() {
	detachAsgInstances.Flags().StringVarP(&ASGName, "name", "n", "", "AWS Autoscaling group name")
	detachAsgInstances.MarkFlagRequired("name")
	Debug.AddCommand(detachAsgInstances)
}
