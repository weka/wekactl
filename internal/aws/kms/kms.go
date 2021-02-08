package kms

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func getKMSTags(clusterName cluster.ClusterName) []*kms.Tag {
	var kmsTags []*kms.Tag
	for key, value := range common.GetCommonTags(clusterName) {
		kmsTags = append(kmsTags, &kms.Tag{
			TagKey:   aws.String(key),
			TagValue: aws.String(value),
		})
	}
	return kmsTags
}

func CreateKMSKey(clusterName cluster.ClusterName, resourceName string) (string, error) {
	svc := connectors.GetAWSSession().KMS

	input := &kms.CreateKeyInput{
		Tags: getKMSTags(clusterName),
	}
	result, err := svc.CreateKey(input)
	if err != nil {
		log.Debug().Msgf(err.Error())
		return "", err
	} else {
		log.Debug().Msgf("KMS key %s was created successfully!", *result.KeyMetadata.KeyId)
		input := &kms.CreateAliasInput{
			AliasName:   aws.String("alias/" + resourceName),
			TargetKeyId: result.KeyMetadata.KeyId,
		}
		_, err := svc.CreateAlias(input)
		if err != nil {
			log.Debug().Msgf(err.Error())
		}
		return *result.KeyMetadata.KeyId, nil
	}
}
