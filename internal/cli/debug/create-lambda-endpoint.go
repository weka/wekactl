package debug

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/apigateway"
	cluster2 "wekactl/internal/aws/cluster"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/iam"
	"wekactl/internal/aws/lambdas"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var createLambdaEndPointCmd = &cobra.Command{
	Use:   "create-lambda-endpoint",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {

			hostGroup := common.HostGroupInfo{
				Name:        "Backends",
				Role:        "backend",
				ClusterName: cluster.ClusterName(StackName),
			}

			functionConfiguration, err := createLambda(hostGroup, lambdas.LambdaJoin, iam.GetJoinAndFetchLambdaPolicy(), lambda.VpcConfig{})
			if err != nil {
				return err
			}

			apiGatewayName := common.GenerateResourceName(hostGroup.ClusterName, hostGroup.Name)
			tags := cluster2.GetHostGroupResourceTags(hostGroup, "v1").AsStringRefs()
			_, err = apigateway.CreateJoinApi(tags, *functionConfiguration.FunctionArn, *functionConfiguration.FunctionName, apiGatewayName, "")
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
	createLambdaEndPointCmd.Flags().StringVarP(&StackName, "name", "n", "", "Cloudformation Stack name")

	_ = createLambdaEndPointCmd.MarkFlagRequired("name")
	_ = createLambdaEndPointCmd.MarkFlagRequired("lambda")
	Debug.AddCommand(createLambdaEndPointCmd)
}
