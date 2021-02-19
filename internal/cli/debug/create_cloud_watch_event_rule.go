package debug

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/cloudwatch"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var StateMachineArn string
var createCloudWatchEventCmd = &cobra.Command{
	Use:   "create-cloud-watch-event-rule",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			hostGroup := hostgroups.HostGroupInfo{
				Name:        "Backends",
				Role:        "backend",
				ClusterName: cluster.ClusterName(StackName),
			}

			roleName := fmt.Sprintf("wekactl-%s-cw-%s", hostGroup.Name, uuid.New().String())
			policyName := fmt.Sprintf("wekactl-%s-cw-%s", string(hostGroup.ClusterName), string(hostGroup.Name))
			assumeRolePolicy := iam.GetCloudWatchEventAssumeRolePolicy()
			policy := iam.GetCloudWatchEventRolePolicy()

			iamTargetVersion := policy.VersionHash()
			iamTags := iam.GetIAMTags(hostGroup, iamTargetVersion)
			roleArn, err := iam.CreateIamRole(iamTags, roleName, policyName, assumeRolePolicy, policy)
			if err != nil {
				return err
			}

			ruleName := common.GenerateResourceName(hostGroup.ClusterName, hostGroup.Name)
			cloudwatchTags := cloudwatch.GetCloudWatchEventTags(hostGroup, "v1")
			err = cloudwatch.CreateCloudWatchEventRule(cloudwatchTags, &StateMachineArn, *roleArn, ruleName)
			if err != nil {
				return err
			}
			logging.UserSuccess("CloudWatchEvent rule creation completed successfully!")
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
		return nil
	},
}

func init() {
	createCloudWatchEventCmd.Flags().StringVarP(&StackName, "name", "n", "", "StackName")
	createCloudWatchEventCmd.Flags().StringVarP(&StateMachineArn, "arn", "m", "", "StateMachineArn")

	_ = createCloudWatchEventCmd.MarkFlagRequired("name")
	Debug.AddCommand(createCloudWatchEventCmd)
}
