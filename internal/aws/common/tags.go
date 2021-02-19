package common

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
)

type Tags map[string]string
type TagsRefsValues map[string]*string


func (t Tags) ToDynamoDb() (ret []*dynamodb.Tag) {
	for k, v := range t {
		ret = append(ret, &dynamodb.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return
}

func (t Tags) Update(tags Tags) Tags {
	for k, v := range tags {
		t[k] = v
	}
	return t
}

func (t Tags) Clone() Tags {
	newTags := Tags{}
	for k, v := range t {
		newTags[k] = v
	}
	return newTags
}

func (t Tags) AsStringRefs() TagsRefsValues {
	tagsRefs := TagsRefsValues{}
	for key, value := range t {
		v := value
		tagsRefs[key] = &v
	}
	return tagsRefs
}

func GetCommonTags(clusterName cluster.ClusterName, version string) Tags {
	tags := Tags{
		"wekactl.io/managed":      "true",
		"wekactl.io/api_version":  "v1",
		"wekactl.io/version":        version,
		"wekactl.io/cluster_name": string(clusterName),
	}
	return tags
}

func GetHostGroupTags(hostGroup hostgroups.HostGroupInfo, version string) Tags {
	tags := GetCommonTags(hostGroup.ClusterName, version)
	return tags.Update(Tags{
		"wekactl.io/hostgroup_name": string(hostGroup.Name),
		"wekactl.io/hostgroup_type": string(hostGroup.Role),
	})
}
