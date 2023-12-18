package cluster

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"syscall"
	"wekactl/internal/aws/alb"
	"wekactl/internal/aws/cluster"
	cluster2 "wekactl/internal/cluster"
	"wekactl/internal/env"
	"wekactl/internal/logging"
)

var importParams cluster2.ImportParams

var importCmd = &cobra.Command{
	Use:   "import [flags]",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if env.Config.Provider == "aws" {
			if importParams.Password == "" {
				fmt.Print("Please enter your weka cluster password: ")
				bytePassword, _ := term.ReadPassword(syscall.Stdin)
				importParams.Password = string(bytePassword)
				fmt.Println()
			}

			err := cluster.ImportCluster(importParams)
			if err != nil {
				logging.UserFailure("Import failed!")
				return err
			}
			logging.UserSuccess("Import finished successfully!")

			err = alb.PrintStatelessClientsJoinScript(cluster2.ClusterName(importParams.Name), importParams.DnsAlias)
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
	importCmd.Flags().StringVarP(&importParams.Name, "name", "n", "", "weka cluster name")
	importCmd.Flags().StringArrayVarP(&importParams.InstanceIds, "instance-ids", "i", []string{}, "weka cluster instance ids")
	importCmd.Flags().StringVarP(&importParams.Username, "username", "u", "", "cluster admin username")
	importCmd.Flags().StringVarP(&importParams.Password, "password", "p", "", "cluster admin password")
	importCmd.Flags().StringArrayVarP(&importParams.TagsList, "tags", "t", []string{}, "cloud resources tags, each tag should be passed in this pattern: '-t key=value'")
	importCmd.Flags().BoolVarP(&importParams.PrivateSubnet, "private-subnet", "s", false, "cluster runs in private subnet, requires execute-api VPC endpoint to present on VPC")
	importCmd.Flags().StringVarP(&importParams.AdditionalAlbSubnet, "additional-alb-subnet", "a", "", "Additional subnet to use for ALB")
	importCmd.Flags().StringVarP(&importParams.DnsAlias, "dns-alias", "l", "", "ALB dns alias")
	importCmd.Flags().StringVarP(&importParams.DnsZoneId, "dns-zone-id", "z", "", "ALB dns zone id")
	importCmd.Flags().BoolVarP(&importParams.UseDynamoDBEndpoint, "use-dynamodb-endpoint", "d", false, "Use dynamoDB endpoint, this will allow avoiding the need to pass the weka cluster password from fetch lambda to scale down lambda and will not show it on the step function input/output")
	_ = importCmd.MarkFlagRequired("name")
	_ = importCmd.MarkFlagRequired("username")
}
