package common

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/cluster"
)

type Tags map[string]string
type TagsRefsValues map[string]*string

const VersionTagKey = "wekactl.io/version"

func (t Tags) ToDynamoDb() (ret []*dynamodb.Tag) {
	for k, v := range t {
		ret = append(ret, &dynamodb.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return
}

func (t Tags) AsEc2() []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for key, value := range t{
		ec2Tags = append(ec2Tags, &ec2.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return ec2Tags
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


func GetCommonResourceTags(clusterName cluster.ClusterName, resourceVersion string) Tags {
	tags := Tags{
		"wekactl.io/managed":      "true",
		"wekactl.io/api_version":  "v1",
		VersionTagKey:             resourceVersion,
		"wekactl.io/cluster_name": string(clusterName),
	}
	return tags
}

func GetHostGroupResourceTags(hostGroup hostgroups.HostGroupInfo, resourceVersion string) Tags {
	tags := GetCommonResourceTags(hostGroup.ClusterName, resourceVersion)
	return tags.Update(Tags{
		"wekactl.io/hostgroup_name": string(hostGroup.Name),
		"wekactl.io/hostgroup_type": string(hostGroup.Role),
	})
}
