package cluster

/*

- consider import/create recovery
- consider destroy
- consider multiple clouds [OPTIONAL]

*/

type Resource interface {
	CurrentVersion() string
	TargetVersion() string
	Delete() error // Consider if it's public part of interface and if needed at all, assuming we have Update
	Create() error // same
	//IsUpdatable() bool
	Update() error
}

func EnsureResource(r Resource) error {
	if r.CurrentVersion() == "" {
		return r.Create()
	}

	if r.CurrentVersion() != r.TargetVersion() {
		return r.Update()
	}
	return nil
}

func UpdateByRecreate(r Resource) error {
	if err := r.Delete(); err != nil {
		return err
	}
	return r.Create()
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
