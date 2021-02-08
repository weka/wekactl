package debug

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	cluster2 "wekactl/internal/aws/cluster"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/aws/scalemachine"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var createStateMachineCmd = &cobra.Command{
	Use:   "create-state-machine",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			hostGroup := hostgroups.HostGroupInfo{
				Name:        "Backends",
				Role:        "backend",
				ClusterName: cluster.ClusterName(StackName),
			}

			stackInstances, err := cluster2.GetStackInstancesInfo(StackName)
			if err != nil {
				return err
			}
			instance := stackInstances.Backends[0]
			lambdaVpcConfig := lambdas.GetLambdaVpcConfig(*instance.SubnetId, cluster2.GetInstanceSecurityGroupsId(instance))

			fetchLambda, err := createLambda(hostGroup, lambdas.LambdaFetchInfo, iam.GetJoinAndFetchLambdaPolicy(), lambda.VpcConfig{})
			if err != nil {
				return err
			}

			scaleLambda, err := createLambda(hostGroup, lambdas.LambdaScale, iam.GetScaleLambdaPolicy(), lambdaVpcConfig)
			if err != nil {
				return err
			}

			terminateLambda, err := createLambda(hostGroup, lambdas.LambdaTerminate, iam.GetTerminateLambdaPolicy(), lambda.VpcConfig{})
			if err != nil {
				return err
			}

			transientLambda, err := createLambda(hostGroup, lambdas.LambdaTransient, iam.PolicyDocument{}, lambda.VpcConfig{})
			if err != nil {
				return err
			}

			lambdas := scalemachine.StateMachineLambdasArn{
				Fetch:     *fetchLambda.FunctionArn,
				Scale:     *scaleLambda.FunctionArn,
				Terminate: *terminateLambda.FunctionArn,
				Transient: *transientLambda.FunctionArn,
			}

			roleName := fmt.Sprintf("wekactl-%s-sm-%s", hostGroup.Name, uuid.New().String())
			policyName := fmt.Sprintf("wekactl-%s-sm-%s", string(hostGroup.ClusterName), string(hostGroup.Name))
			assumeRolePolicy := iam.GetStateMachineAssumeRolePolicy()
			policy := iam.GetStateMachineRolePolicy()

			roleArn, err := iam.CreateIamRole(hostGroup, roleName, policyName, assumeRolePolicy, policy)
			if err != nil {
				return err
			}

			stateMachineName := common.GenerateResourceName(hostGroup.ClusterName, hostGroup.Name)
			_, err = scalemachine.CreateStateMachine(hostGroup, lambdas, *roleArn, stateMachineName)
			if err != nil {
				return err
			}
			logging.UserSuccess("State machine creation completed successfully!")
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
		return nil
	},
}

func init() {
	createStateMachineCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")

	_ = createStateMachineCmd.MarkFlagRequired("name")
	Debug.AddCommand(createStateMachineCmd)
}
