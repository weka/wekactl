package hostgroup

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/hostgroup"
)

var CreateCmd = &cobra.Command{
	Use:   "create [flags]",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if Provider == "aws" {
			name := hostgroup.CreateAutoScalingGroup(Region, InstanceId, MinSize, MaxSize)
			fmt.Printf("HostGroup: \"%s\" was created successfully!\n", name)
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", Provider)
		}
	},
}

func init() {
	CreateCmd.Flags().StringVarP(&Provider, "provider", "p", "aws", "Cloud provider")
	CreateCmd.Flags().StringVarP(&Region, "region", "r", "", "Region")
	CreateCmd.Flags().StringVarP(&InstanceId, "instanceId", "i", "", "Instance id")
	CreateCmd.Flags().Int64VarP(&MinSize, "min-size", "", 0, "Autoscaling group minimum size")
	CreateCmd.Flags().Int64VarP(&MaxSize, "max-size", "", 0, "Autoscaling group maximum size")
	CreateCmd.MarkFlagRequired("instanceId")

	HostGroup.AddCommand(CreateCmd)
}
