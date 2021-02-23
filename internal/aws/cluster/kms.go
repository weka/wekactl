package cluster

import (
	awsKms "github.com/aws/aws-sdk-go/service/kms"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/kms"
	"wekactl/internal/cluster"
)

const kmsVersion = "v1"

type KmsKey struct {
	Key         string
	Version     string
	ClusterName cluster.ClusterName
}

func (k *KmsKey) Tags() interface{} {
	return kms.GetKMSTags(k.ClusterName, k.TargetVersion())
}

func (k *KmsKey) SubResources() []cluster.Resource {
	return []cluster.Resource{}
}

func (k *KmsKey) ResourceName() string {
	return common.GenerateResourceName(k.ClusterName, "")
}

func (k *KmsKey) Fetch() error {
	version, err := kms.GetKMSKeyVersion(k.ClusterName)
	if err != nil {
		return err
	}
	k.Version = version
	return nil
}

func (k *KmsKey) Init() {
	return
}

func (k *KmsKey) DeployedVersion() string {
	return k.Version
}

func (k *KmsKey) TargetVersion() string {
	return kmsVersion
}

func (k *KmsKey) Delete() error {
	return kms.DeleteKMSKey(k.ResourceName(), k.ClusterName)
}

func (k *KmsKey) Create() error {
	kmsKey, err := kms.CreateKMSKey(k.Tags().([]*awsKms.Tag), k.ResourceName())
	if err != nil {
		return err
	}

	k.Key = kmsKey
	return nil
}

func (k *KmsKey) Update() error {
	err := k.Delete()
	if err!=nil{
		return err
	}
	return k.Create()
}
