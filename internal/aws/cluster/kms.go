package cluster

import (
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/kms"
	"wekactl/internal/cluster"
)

type KmsKey struct {
	Key         string
	ClusterName cluster.ClusterName
}

func (k *KmsKey) ResourceName() string {
	return common.GenerateResourceName(k.ClusterName, "")
}

func (k *KmsKey) Fetch() error {
	return nil
}

func (k *KmsKey) Init() {
	return
}

func (k *KmsKey) DeployedVersion() string {
	return ""
}

func (k *KmsKey) TargetVersion() string {
	return ""
}

func (k *KmsKey) Delete() error {
	return kms.DeleteKMSKey(k.ResourceName(), k.ClusterName)
}

func (k *KmsKey) Create() error {
	kmsKey, err := kms.CreateKMSKey(k.ClusterName, k.ResourceName())
	if err != nil {
		return err
	}

	k.Key = kmsKey
	return nil
}

func (k *KmsKey) Update() error {
	panic("implement me")
}
