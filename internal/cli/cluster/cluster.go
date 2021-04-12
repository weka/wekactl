package cluster

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"strings"
	"wekactl/internal/aws/db"
	cluster2 "wekactl/internal/cluster"
)

var Region string
var StackName string
var Tags []string

func generateClusterSettings(tagsList []string) (clusterSettings cluster2.ClusterSettings, err error){
	clusterSettings.Key = db.ModelClusterSettings
	tags := make(cluster2.Tags)
	if len(Tags) > 0 {
		for _, tag := range tagsList {
			keyVal := strings.Split(tag, "=")
			if len(keyVal) != 2 {
				err = errors.New(fmt.Sprintf("Invalid tag %s", tag))
				return
			}
			tags[keyVal[0]] = keyVal[1]
		}
	}
	clusterSettings.Tags = tags

	return
}

var Cluster = &cobra.Command{
	Use:   "cluster [command] [flags]",
	Short: "Cluster operations",
	Run: func(c *cobra.Command, _ []string) {
		if err := c.Help(); err != nil {
			log.Debug().Msgf("ignoring cobra error %q", err.Error())
		}
	},
	SilenceUsage: true,
	Aliases:      []string{"clusters"},
}

func init() {
	Cluster.AddCommand(createCmd)
	Cluster.AddCommand(importCmd)
	Cluster.AddCommand(listCmd)
	Cluster.AddCommand(destroyCmd)
	Cluster.AddCommand(updateCmd)
	Cluster.AddCommand(changeCredentialsCmd)
	_ = Cluster.MarkPersistentFlagRequired("region")
}
