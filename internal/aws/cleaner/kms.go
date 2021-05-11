package cleaner

import (
	"github.com/aws/aws-sdk-go/service/kms"
	kms2 "wekactl/internal/aws/kms"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type KmsKey struct {
	Key         *kms.KeyListEntry
	ClusterName cluster.ClusterName
}

func (k *KmsKey) Fetch() error {
	kmsKey, err := kms2.GetClusterKMSKey(k.ClusterName)
	if err != nil {
		return err
	}
	k.Key = kmsKey
	return nil
}

func (k *KmsKey) Delete() error {
	return kms2.DeleteKmsKey(k.Key, k.ClusterName)
}

func (k *KmsKey) Print() {
	logging.UserInfo("KmsKey:")
	if k.Key != nil {
		logging.UserInfo("\t- %s", *k.Key.KeyId)
	}
}
