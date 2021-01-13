package protocol

import (
	"time"
	"wekactl/internal/lib/weka"
)

type HostGroupInfoResponse struct {
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	PrivateIps      []string `json:"private_ips"`
	DesiredCapacity int      `json:"desired_capacity"`
	InstanceIds     []string `json:"instance_ids"`
	Role            string   `json:"role"`
}

type ScaleResponseHost struct {
	InstanceId string      `json:"instance_id"`
	State      string      `json:"status"`
	AddedTime  time.Time   `json:"added_time"`
	HostId     weka.HostId `json:"host_id"`
}

type ScaleResponse struct {
	Hosts           []ScaleResponseHost `json:"hosts"`
	TransientErrors []string
}

func (r *ScaleResponse) AddTransientErrors(errs []error) {
	for _, err := range errs {
		r.TransientErrors = append(r.TransientErrors, err.Error())
	}
}

type TerminatedInstance struct {
	InstanceId string    `json:"instance_id"`
	Creation   time.Time `json:"creation_date"`
}
type TerminatedInstancesResponse struct {
	Instances       []TerminatedInstance `json:"set_to_terminate_instances"`
	TransientErrors []string
}

func (r *TerminatedInstancesResponse) AddTransientErrors(errs []error) {
	for _, err := range errs {
		r.TransientErrors = append(r.TransientErrors, err.Error())
	}
}
