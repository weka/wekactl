package cluster

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/db"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func createDb(clusterName cluster.ClusterName, stackId string) (tableName string, err error){
	kmsKey, err := createKMSKey(stackId, clusterName)
	if err != nil {
		return
	}

	tableName = generateResourceName(stackId, clusterName, "")
	err = db.CreateDb(tableName, kmsKey, getCommonTags(clusterName).Update(common.Tags{
		"wekactl.io/stack_id": stackId,
	}))
	return
}


func saveCredentials(tableName string, username, password string) error {
	err := db.PutItem(tableName, db.ClusterCreds{
		Key:      db.ModelClusterCreds,
		Username: username,
		Password: password,
	})
	if err != nil {
		log.Debug().Msgf("error saving creds to DB %v", err)
		return err
	}
	log.Debug().Msgf("Username:%s and Password:%s were added to DB successfully!", username, strings.Repeat("*", len(password)))
	return nil
}


func saveClusterParams(tableName string, params db.DefaultClusterParams) error {
	if params.Key == ""{
		params.Key = db.ModelDefaultClusterParams
	}
	err := db.PutItem(tableName, params)
	if err != nil {
		log.Debug().Msgf("error saving cluster params to DB %v", err)
		return err
	}
	return nil
}


func createKMSKey(stackId string, clusterName cluster.ClusterName) (string, error) {
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
		alias := generateResourceName(stackId, clusterName, "")
		input := &kms.CreateAliasInput{
			AliasName:   aws.String("alias/" + alias),
			TargetKeyId: result.KeyMetadata.KeyId,
		}
		_, err := svc.CreateAlias(input)
		if err != nil {
			log.Debug().Msgf(err.Error())
		}
		return *result.KeyMetadata.KeyId, nil
	}
}
