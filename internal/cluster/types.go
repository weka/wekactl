package cluster

//goland:noinspection GoNameStartsWithPackageName
type ClusterName string

type IClusterSettings interface {
	Tags() Tags
	UsePrivateSubnet() bool
}
