package cluster

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type StatementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

func (p PolicyDocument) Bytes() []byte {
	if p.Version == "" {
		p.Version = p.VersionHash()
	}
	policy, err := json.Marshal(&p)
	if err != nil {
		panic(err)
	}
	return policy
}

func (p PolicyDocument) String() string {
	return string(p.Bytes())
}

func (p PolicyDocument) VersionHash() string {
	policy, err := json.Marshal(&p.Statement)
	if err != nil {
		panic(err)
	}
	h := sha256.New()
	h.Write(policy)
	return hex.EncodeToString(h.Sum(nil))
}


type Principal struct {
	Service string
}

//Resource is prohibited for assume role
type PolicyStatement struct {
	Effect    string
	Action    []string
	Principal Principal
}

type AssumeRolePolicyDocument struct {
	Version   string
	Statement []PolicyStatement
}

type IamProfile struct {
	Policy PolicyDocument
}

func (i IamProfile) Fetch() error {
	panic("implement me")
}

func (i IamProfile) Init() {
	panic("implement me")
}

func (i IamProfile) DeployedVersion() string {
	panic("implement me")
}

func (i IamProfile) TargetVersion() string {
	return i.Policy.VersionHash()
}

func (i IamProfile) Delete() error {
	panic("implement me")
}

func (i IamProfile) Create() error {
	panic("implement me")
}

func (i IamProfile) Update() error {
	panic("implement me")
}

