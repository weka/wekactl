package debug

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"strings"
	cluster2 "wekactl/internal/aws/cluster"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	strings2 "wekactl/internal/lib/strings"
	"wekactl/internal/logging"
)

func generateLambdaName(lambdaType lambdas.LambdaType) string {
	n := strings.Join([]string{"wekactl", StackName, string(lambdaType), "Backends"}, "-")
	return strings2.ElfHashSuffixed(n, 64)
}

func createLambda(hostGroup hostgroups.HostGroupInfo, lambdaType lambdas.LambdaType, policy iam.PolicyDocument, vpcConfig lambda.VpcConfig) (functionConfiguration *lambda.FunctionConfiguration, err error) {
	assumeRolePolicy := iam.GetLambdaAssumeRolePolicy()
	roleName := fmt.Sprintf("wekactl-%s-%s-%s", hostGroup.Name, string(lambdaType), uuid.New().String())
	lambdaName := generateLambdaName(lambdaType)
	policyName := lambdaName
	roleArn, err := iam.CreateIamRole(hostGroup, roleName, policyName, assumeRolePolicy, policy)
	if err != nil {
		return
	}
	functionConfiguration, err = lambdas.CreateLambda(hostGroup, lambdaType, lambdaName, *roleArn, vpcConfig)
	if err != nil {
		return
	}
	return
}

var createLambdaCmd = &cobra.Command{
	Use:   "create-lambda",
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
			var lambdaVpcConfig lambda.VpcConfig
			var lambdaType lambdas.LambdaType
			var policy iam.PolicyDocument
			switch Lambda {
			case "join":
				policy = iam.GetJoinAndFetchLambdaPolicy()
				lambdaType = lambdas.LambdaJoin
			case "fetch":
				policy = iam.GetJoinAndFetchLambdaPolicy()
				lambdaType = lambdas.LambdaFetchInfo
			case "scale":
				policy = iam.GetScaleLambdaPolicy()
				lambdaType = lambdas.LambdaScale
				instance := stackInstances.Backends[0]
				lambdaVpcConfig = lambdas.GetLambdaVpcConfig(*instance.SubnetId, cluster2.GetInstanceSecurityGroupsId(instance))
			case "terminate":
				policy = iam.GetTerminateLambdaPolicy()
				lambdaType = lambdas.LambdaTerminate
			default:
				err = errors.New("invalid lambda type")
			}

			if err != nil {
				return err
			}

			_, err = createLambda(hostGroup, lambdaType, policy, lambdaVpcConfig)
			if err != nil {
				return err
			}
			logging.UserSuccess("Lambda endpoint creation completed successfully!")
		} else {
			fmt.Printf("Cloud provider '%s' is not supported with this action\n", env.Config.Provider)
		}
		return nil
	},
}

func init() {
	createLambdaCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")
	createLambdaCmd.Flags().StringVarP(&Lambda, "lambda", "t", "", "Lambda type to create")

	_ = createLambdaCmd.MarkFlagRequired("name")
	_ = createLambdaCmd.MarkFlagRequired("lambda")
	Debug.AddCommand(createLambdaCmd)
}
