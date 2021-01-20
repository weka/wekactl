package cluster

const joinApiVersion ="v1"
type ApiGateway struct {
	HgInfo HostGroupInfo
	backend         Lambda
	deployedVersion string
}

func (a *ApiGateway) Init() {
	a.backend.HgInfo = a.HgInfo
	a.backend.Permissions = []StatementEntry{
		{
			Effect: "Allow",
			Action: []string{
				"logs:CreateLogStream",
				"logs:PutLogEvents",
				"logs:CreateLogGroup",
				"dynamodb:GetItem",
				"autoscaling:Describe*",
				"ec2:Describe*",
				"kms:Decrypt",
			},
			Resource: "*",
		},
	}
	a.backend.Init()
}


func (a *ApiGateway) Fetch() error {
	panic("implement me")
}

func (a *ApiGateway) DeployedVersion() string {
	return a.deployedVersion
}

func (a *ApiGateway) TargetVersion() string {
	return joinApiVersion + a.backend.TargetVersion()
}

func (a *ApiGateway) Delete() error {
	panic("implement me")
}

func (a *ApiGateway) Create() error {
	panic("implement me")
}

func (a *ApiGateway) Update() error {
	if a.DeployedVersion() == a.TargetVersion(){
		return nil
	}
	err := a.backend.Update()
	if err != nil {
		return err
	}
	return nil
}

