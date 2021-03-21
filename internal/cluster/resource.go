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
	Tags() Tags
	Fetch() error
	DeployedVersion() string
	TargetVersion() string
	Delete() error
	Create() error
	Update() error
	Init()
}

func EnsureResource(r Resource) error {
	for _, subresource := range r.SubResources() {
		if err := EnsureResource(subresource); err != nil {
			return err
		}
	}

	resourceType := strings.TrimLeft(reflect.TypeOf(r).String(), "*cluster.")

	err := r.Fetch()
	if err != nil {
		return err
	}

	if r.DeployedVersion() == "" {
		log.Debug().Msgf("creating resource %s %s ...", resourceType, r.ResourceName())
		return r.Create()
	}

	if r.DeployedVersion() != r.TargetVersion() {
		log.Debug().Msgf("updating resource %s %s ...", resourceType, r.ResourceName())
		return r.Update()
	}

	log.Debug().Msgf("resource %s %s exists and updated", resourceType, r.ResourceName())
	return nil
}

func DestroyResource(r Resource) error {
	for _, subresource := range r.SubResources() {
		if err := DestroyResource(subresource); err != nil {
			return err
		}
	}

	return r.Delete()
}
