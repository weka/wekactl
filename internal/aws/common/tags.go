package common

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Tags map[string]string

func (t Tags) ToDynamoDb() (ret []*dynamodb.Tag) {
	for k, v := range t{
		ret = append(ret, &dynamodb.Tag{
			Key:   &k,
			Value: &v,
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


