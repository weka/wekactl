package common

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
)

type Tags map[string]string

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

func GetCommonTags(clusterName cluster.ClusterName) Tags {
	tags := Tags{
		"wekactl.io/managed":      "true",
		"wekactl.io/api_version":  "v1",
		"wekactl.io/cluster_name": string(clusterName),
	}
	return tags
}

func GetMapCommonTags(hostGroup hostgroups.HostGroupInfo) map[string]*string {
	return map[string]*string{
		"wekactl.io/managed":        aws.String("true"),
		"wekactl.io/api_version":    aws.String("v1"),
		"wekactl.io/cluster_name":   aws.String(string(hostGroup.ClusterName)),
		"wekactl.io/hostgroup_name": aws.String(string(hostGroup.Name)),
		"wekactl.io/hostgroup_type": aws.String(string(hostGroup.Role)),
	}
}

func GetHostGroupTags(hostGroup hostgroups.HostGroupInfo) Tags {
	tags := GetCommonTags(hostGroup.ClusterName)
	return tags.Update(Tags{
		"wekactl.io/hostgroup_name": string(hostGroup.Name),
		"wekactl.io/hostgroup_type": string(hostGroup.Role),
	})
}
