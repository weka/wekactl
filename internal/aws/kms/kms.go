package kms

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func GetKmsAliasName(clusterName cluster.ClusterName) string {
	return common.GenerateResourceName(clusterName, "")
}

func CreateKMSKey(tags []*kms.Tag, resourceName string) (string, error) {
	svc := connectors.GetAWSSession().KMS

	input := &kms.CreateKeyInput{
		Tags: tags,
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

func GetKmsKey(clusterName cluster.ClusterName) (*kms.KeyListEntry, error) {
	svc := connectors.GetAWSSession().KMS
	var keyListEntry *kms.KeyListEntry

	kmsKeysOutput, err := svc.ListKeys(&kms.ListKeysInput{})
	if err != nil {
		return keyListEntry, err
	}

	for _, kmsKey := range kmsKeysOutput.Keys {
		keyKeyInfo, err := svc.DescribeKey(&kms.DescribeKeyInput{KeyId: kmsKey.KeyId})
		if err != nil {
			return keyListEntry, err
		}
		if *keyKeyInfo.KeyMetadata.KeyState == kms.KeyStatePendingDeletion || *keyKeyInfo.KeyMetadata.KeyManager == kms.KeyManagerTypeAws {
			continue
		}

		tags, err := svc.ListResourceTags(&kms.ListResourceTagsInput{KeyId: kmsKey.KeyId})
		if err != nil {
			return keyListEntry, err
		}
		isClusterKey := false
		for _, tag := range tags.Tags {
			if *tag.TagValue == string(clusterName) {
				isClusterKey = true
				break
			}
		}

		if isClusterKey {
			keyListEntry = kmsKey
			break
		}
	}
	return keyListEntry, nil
}

func DeleteKMSKey(aliasName string, clusterName cluster.ClusterName) error {
	svc := connectors.GetAWSSession().KMS

	_, err := svc.DeleteAlias(&kms.DeleteAliasInput{
		AliasName: aws.String("alias/" + aliasName),
	})
	if err != nil {
		if _, ok := err.(*kms.NotFoundException); !ok {
			return err
		}
	} else {
		log.Debug().Msgf("kms alias alias/%s was deleted successfully", aliasName)
	}

	kmsKey, err := GetKmsKey(clusterName)
	if err != nil {
		return err
	}
	if kmsKey == nil {
		return nil
	}

	_, err = svc.ScheduleKeyDeletion(&kms.ScheduleKeyDeletionInput{
		KeyId:               kmsKey.KeyId,
		PendingWindowInDays: aws.Int64(7),
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("kms key %s was deleted successfully", *kmsKey.KeyArn)

	return nil
}

func GetKMSKeyVersion(clusterName cluster.ClusterName) (version string, err error) {
	svc := connectors.GetAWSSession().KMS
	kmsKey, err := GetKmsKey(clusterName)
	if err != nil || kmsKey == nil {
		return
	}

	tagsOutput, err := svc.ListResourceTags(&kms.ListResourceTagsInput{KeyId: kmsKey.KeyId})
	if err != nil {
		return
	}

	for _, tag := range tagsOutput.Tags {
		if *tag.TagKey == cluster.VersionTagKey {
			version = *tag.TagValue
			return
		}
	}
	return
}

func GetKMSKeyId(clusterName cluster.ClusterName) (arn string, err error) {
	kmsKey, err := GetKmsKey(clusterName)
	if err != nil || kmsKey == nil {
		return
	}
	arn = *kmsKey.KeyId
	return
}

func GetClusterKMSKey(clusterName cluster.ClusterName) (kmsKey *kms.KeyListEntry, err error) {
	svc := connectors.GetAWSSession().KMS

	kmsKeysOutput, err := svc.ListKeys(&kms.ListKeysInput{})
	if err != nil {
		return
	}

	var keyKeyInfo *kms.DescribeKeyOutput
	var tags *kms.ListResourceTagsOutput
	for _, key := range kmsKeysOutput.Keys {
		keyKeyInfo, err = svc.DescribeKey(&kms.DescribeKeyInput{KeyId: key.KeyId})
		if err != nil {
			return
		}
		if *keyKeyInfo.KeyMetadata.KeyState == kms.KeyStatePendingDeletion || *keyKeyInfo.KeyMetadata.KeyManager == kms.KeyManagerTypeAws {
			continue
		}

		tags, err = svc.ListResourceTags(&kms.ListResourceTagsInput{KeyId: key.KeyId})
		if err != nil {
			return
		}

		for _, tag := range tags.Tags {
			if *tag.TagKey == cluster.ClusterNameTagKey && *tag.TagValue == string(clusterName) {
				return key, nil
			}
		}

	}
	return
}

func DeleteKmsKey(kmsKey *kms.KeyListEntry, clusterName cluster.ClusterName) error {
	if kmsKey != nil {
		err := DeleteKMSKey(GetKmsAliasName(clusterName), clusterName)
		if err != nil {
			return err
		}
	}
	return nil
}
