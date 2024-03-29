package debug

import (
	"fmt"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/debug"
	"wekactl/internal/env"
)

var InstanceId string
var MinSize int64
var MaxSize int64

var createAutoScalingGroupCmd = &cobra.Command{
	Use:   "create-auto-scaling-group",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if env.Config.Provider == "aws" {
			debug.CreateAutoScalingGroup(InstanceId, MinSize, MaxSize)
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
	},
}

func init() {
	createAutoScalingGroupCmd.Flags().StringVarP(&InstanceId, "instanceId", "i", "", "instance id")
	createAutoScalingGroupCmd.Flags().Int64VarP(&MinSize, "min-size", "", 0, "min size")
	createAutoScalingGroupCmd.Flags().Int64VarP(&MaxSize, "max-size", "", 0, "max size")
	_ = createAutoScalingGroupCmd.MarkFlagRequired("instanceId")
	Debug.AddCommand(createAutoScalingGroupCmd)
}
