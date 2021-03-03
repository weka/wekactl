package cluster

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sfn"
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

func (t Tags) AsAsg() []*autoscaling.Tag {
	var autoscalingTags []*autoscaling.Tag
	for key, value := range t {
		autoscalingTags = append(autoscalingTags, &autoscaling.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return autoscalingTags
}

func (t Tags) AsCloudWatch() []*cloudwatchevents.Tag {
	var cloudWatchEventTags []*cloudwatchevents.Tag
	for key, value := range t {
		cloudWatchEventTags = append(cloudWatchEventTags, &cloudwatchevents.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return cloudWatchEventTags
}

func (t Tags) AsIam() []*iam.Tag {
	var iamTags []*iam.Tag
	for key, value := range t {
		iamTags = append(iamTags, &iam.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return iamTags
}

func (t Tags) AsKms() []*kms.Tag {
	var kmsTags []*kms.Tag
	for key, value := range t {
		kmsTags = append(kmsTags, &kms.Tag{
			TagKey:   aws.String(key),
			TagValue: aws.String(value),
		})
	}
	return kmsTags
}

func (t Tags) AsEc2() []*ec2.Tag {
	var ec2Tags []*ec2.Tag
	for key, value := range t {
		ec2Tags = append(ec2Tags, &ec2.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return ec2Tags
}

func (t Tags) AsSfn() []*sfn.Tag {
	var sfnTags []*sfn.Tag
	for key, value := range t {
		sfnTags = append(sfnTags, &sfn.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return sfnTags
}

func GetResourceVersionTag(version string) Tags {
	return Tags{
		VersionTagKey: version,
	}
}

func GetCommonResourceTags(clusterName ClusterName, version string) Tags {
	tags := Tags{
		"wekactl.io/managed":      "true",
		"wekactl.io/api_version":  "v1",
		VersionTagKey:             version,
		"wekactl.io/cluster_name": string(clusterName),
	}
	return tags
}
