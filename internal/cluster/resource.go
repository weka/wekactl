package cluster

import (
	"github.com/rs/zerolog/log"
	"reflect"
	"strings"
	"wekactl/internal/logging"
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
	Create(tags Tags) error
	Update() error
	Init()
	//UpdateTags(tags Tags) error
}

func EnsureResource(r Resource, clusterSettings IClusterSettings, dryRun bool) error {
	for _, subresource := range r.SubResources() {
		if err := EnsureResource(subresource, clusterSettings, dryRun); err != nil {
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
		if resourceType == "HostGroup" || resourceType == "AWSCluster" {
			// these resources are not actual aws resources, so we want to log them only to developers
			log.Debug().Msgf("creating resource %s %s ...", resourceType, r.ResourceName())
		} else {
			if dryRun {
				logging.UserInfo("resource %s \"%s\" will be created", resourceType, r.ResourceName())
				return nil
			}
			log.Info().Msgf("creating resource %s %s ...", resourceType, r.ResourceName())
		}
		return r.Create(tags)
	}

	if r.DeployedVersion() != r.TargetVersion() {
		if dryRun {
			logging.UserInfo("resource %s \"%s\" will be updated", resourceType, r.ResourceName())
			return nil
		}
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
