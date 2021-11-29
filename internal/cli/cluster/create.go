package cluster

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/spf13/cobra"
	"wekactl/internal/aws/stack"
	"wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var createParams cluster.CreateParams

var createCmd = &cobra.Command{
	Use:   "create [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider != "aws" {
			err := errors.New(fmt.Sprintf("Cloud provider '%s' is not supported with this action", env.Config.Provider))
			return err
		}

		err := stack.CreateStack(createParams)
		if err != nil {
			return err
		}
		logging.UserSuccess("Cloud formation stack was created, waiting for it to be ready, approximately ~10M.")
		status, err := stack.WaitForStackCreationComplete(createParams.Name)
		if err != nil {
			return err
		}

		if status != cloudformation.StackStatusCreateComplete {
			logging.UserSuccess("Cloud formation stack wasn't complete, ended with status:", status)
		}

		importParams.Name = createParams.Name
		return ImportClusterAndPrintJoinScript(importParams)
	},
}

func init() {
	createCmd.Flags().StringVarP(&createParams.Name, "name", "n", "", "weka cluster name")
	createCmd.Flags().StringVarP(&createParams.VpcId, "vpcId", "v", "", "weka cluster vpc id")
	createCmd.Flags().StringVarP(&createParams.SubnetId, "subnetId", "s", "", "weka cluster subnet id")
	createCmd.Flags().StringVarP(&createParams.NetworkTopology, "networkTopology", "", "Public subnet", "weka cluster network topology")
	createCmd.Flags().StringVarP(&createParams.CustomProxy, "customProxy", "", "", "weka cluster custom proxy")
	createCmd.Flags().StringVarP(&createParams.KeyName, "keyName", "k", "", "weka cluster key name")
	createCmd.Flags().StringVarP(&createParams.Token, "token", "a", "", "weka cluster api token")
	createCmd.Flags().StringVarP(&createParams.InstanceType, "instance-type", "i", "", "weka cluster instance type")
	createCmd.Flags().StringVarP(&createParams.WekaVersion, "weka-version", "w", "", "weka cluster weka version")
	createCmd.Flags().IntVarP(&createParams.Count, "count", "c", -1, "weka cluster number of instances")

	createCmd.Flags().StringVarP(&importParams.Username, "username", "u", "admin", "cluster admin username")
	createCmd.Flags().StringVarP(&importParams.Password, "password", "p", "admin", "cluster admin password")
	createCmd.Flags().StringArrayVarP(&importParams.TagsList, "tags", "t", []string{}, "cloud resources tags, each tag should be passed in this pattern: '-t key=value'")
	createCmd.Flags().BoolVarP(&importParams.PrivateSubnet, "private-subnet", "", false, "cluster runs in private subnet, requires execute-api VPC endpoint to present on VPC")
	createCmd.Flags().StringVarP(&importParams.AdditionalAlbSubnet, "additional-alb-subnet", "", "", "Additional subnet to use for ALB")
	createCmd.Flags().StringVarP(&importParams.DnsAlias, "dns-alias", "l", "", "ALB dns alias")
	createCmd.Flags().StringVarP(&importParams.DnsZoneId, "dns-zone-id", "z", "", "ALB dns zone id")

	_ = createCmd.MarkFlagRequired("name")
	_ = createCmd.MarkFlagRequired("vpcId")
	_ = createCmd.MarkFlagRequired("subnetId")
	_ = createCmd.MarkFlagRequired("token")
	_ = createCmd.MarkFlagRequired("role")
	_ = createCmd.MarkFlagRequired("instance-type")
	_ = createCmd.MarkFlagRequired("weka-version")
	_ = createCmd.MarkFlagRequired("count")
}
