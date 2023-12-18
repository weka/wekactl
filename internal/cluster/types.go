package cluster

import (
	"github.com/rs/zerolog/log"
	"strings"
)

//goland:noinspection GoNameStartsWithPackageName
type ClusterName string

type IClusterSettings interface {
	Tags() Tags
}

type ImportParams struct {
	Name                string
	InstanceIds         []string
	Username            string
	Password            string
	TagsList            []string
	PrivateSubnet       bool
	AdditionalAlbSubnet string
	DnsAlias            string
	DnsZoneId           string
	UseDynamoDBEndpoint bool
}

func (params ImportParams) TagsMap() Tags {
	tags := make(Tags)
	if len(params.TagsList) > 0 {
		for _, tag := range params.TagsList {
			keyVal := strings.Split(tag, "=")
			if len(keyVal) != 2 {
				log.Fatal().Msgf("=(equal sign) is not allowed both in keys and values")
			}
			tags[keyVal[0]] = keyVal[1]
		}
	}
	return tags
}
