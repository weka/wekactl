package cluster

/*

- consider import/create recovery
- consider destroy
- consider multiple clouds [OPTIONAL]

*/

type Resource interface {
	Fetch() error
	DeployedVersion() string
	TargetVersion() string
	Delete() error
	Create() error
	Update() error
	Init()
}

func EnsureResource(r Resource) error {
	r.Fetch()
	if r.DeployedVersion() == "" {
		return r.Create()
	}

	if r.DeployedVersion() != r.TargetVersion() {
		return r.Update()
	}
	return nil
}

/*
struct AWSStack{
	AWSHostGroup: []HostGroup<Resource>
	DynamoDB: DynamoDb<Resource>
}

struct DynamoDb{
	KMSKey: KmsKey<Resource>
}

struct AWSHostGroup{
	PeriodicScale<Resource>
	Autoscaling<Resource>
}

struct Autoscaling<Resource>{
	LaunchTemplate<Resource>
}

struct LaunchTemplate {
	SecurityGroup
	IAMRole
	JoinAPI<Resource>
}

struct PeriodicScale<Resource>{
	StateMachine<Resource>
}

struct StateMachine<Resource>{
	Lambdas: []Lambda<Resource>
}

struct Lambda<Resource>{
	Role<Resource>
}

struct JoinAPI<Resource>==ApiGateway{
	Lambda<Resource>
}

*/
