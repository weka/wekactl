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
	Fetch() error
	DeployedVersion() string
	TargetVersion() string
	Delete() error
	Create() error
	Update() error
	Init()
}

func EnsureResource(r Resource) error {
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
