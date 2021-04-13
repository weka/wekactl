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
	Create(tags Tags) error
	Update() error
	Init()
	//UpdateTags(tags Tags) error
}

func EnsureResource(r Resource, clusterSettings IClusterSettings) error {
	for _, subresource := range r.SubResources() {
		if err := EnsureResource(subresource, clusterSettings); err != nil {
			return err
		}
	}

	resourceType := strings.TrimLeft(reflect.TypeOf(r).String(), "*cluster.")
	tags := r.Tags().Update(clusterSettings.Tags())

	err := r.Fetch()
	if err != nil {
		return err
	}

	if r.DeployedVersion() == "" {
		log.Info().Msgf("creating resource %s %s ...", resourceType, r.ResourceName())
		return r.Create(tags)
	}

	if r.DeployedVersion() != r.TargetVersion() {
		log.Info().Msgf("updating resource %s %s ...", resourceType, r.ResourceName())
		return r.Update()
	}

	//if len(clusterSettings.Tags) > 0 {
	//	log.Info().Msgf("updating resource %s %s ...", resourceType, r.ResourceName())
	//	return r.UpdateTags(tags)
	//}

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
