package cluster

import (
	"math/rand"
	"strings"
	"wekactl/internal/aws/dist"
	"wekactl/internal/lib/math"
	strings2 "wekactl/internal/lib/strings"
)

type LambdaType string

const LambdaFetchInfo LambdaType = "Fetch"
const LambdaScale LambdaType = "Scale"
const LambdaTerminate LambdaType = "terminate"
const LambdaJoin LambdaType = "Join"


type Lambda struct {
	Name string
	Type         LambdaType
	Profile      IamProfile
	HgInfo       HostGroupInfo
	Permissions   []StatementEntry
}

func (l *Lambda) resourceName() string {
	n := strings.Join([]string{"wekactl", string(l.HgInfo.ClusterName), l.Name, string(l.HgInfo.Name)}, "-")
	return strings2.ElfHashSuffixed(n, 64)
}

func (l *Lambda) Fetch() error {
	//searchTags := getHostGroupTags(l.HgInfo).Update()
	//panic("implement me")
}

func (l *Lambda) Init() {
	l.Profile.Policy.Statement = l.Permissions
	l.Profile.Init()
}

func (l *Lambda) DeployedVersion() string {
	panic("implement me")
}

func (l *Lambda) TargetVersion() string {
	return dist.LambdasID + l.Profile.TargetVersion()
}

func (l *Lambda) Delete() error {
	panic("implement me")
}

func (l *Lambda) Create() error {
	panic("implement me")
}

func (l *Lambda) Update() error {
	if l.DeployedVersion() == l.TargetVersion() {
		return nil
	}
	err := l.Profile.Update()
	if err != nil {
		return err
	}
	return nil
}
