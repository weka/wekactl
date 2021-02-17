package cluster

import (
	"github.com/rs/zerolog/log"
	"reflect"
	"strings"
)

/*

- consider import/create recovery
- consider destroy
- consider multiple clouds [OPTIONAL]

*/

type Resource interface {
	ResourceName() string
	SubResources() []Resource
	Fetch() error
	DeployedVersion() string
	TargetVersion() string
	Delete() error
	Create() error
	Update() error
	Init()
	//TODO: Probably Add Tags, finish with APIGateway to pass HostGroup -> Tags
}

//func EnsureResource(r Resource) error {
//	err := r.Fetch()
//	if err != nil {
//		return err
//	}
//	if r.DeployedVersion() == "" {
//		return r.Create()
//	}
//
//	return r.Update()
//	//if r.DeployedVersion() != r.TargetVersion() {
//	//}
//	//
//	//log.Debug().Msgf("%s resource exists and updated", strings.Trim(reflect.TypeOf(r).String(), "*cluster."))
//
//	//return nil
//}
//

func EnsureResource(r Resource) error {
	for _, subresource := range r.SubResources() {
		if err := EnsureResource(subresource); err != nil {
			return err
		}
	}

	err := r.Fetch()
	if err != nil {
		return err
	}

	if r.DeployedVersion() == "" {
		return r.Create()
	}

	if r.DeployedVersion() != r.TargetVersion() {
		return r.Update()
	}
	log.Debug().Msgf("%s resource exists and updated", strings.Trim(reflect.TypeOf(r).String(), "*cluster."))
	return nil
}
