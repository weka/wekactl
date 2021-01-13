package weka

import (
	"github.com/google/uuid"
	"time"
)

type JrpcMethod string

const (
	JrpcHostList         JrpcMethod = "hosts_list"
	JrpcDrivesList       JrpcMethod = "disks_list"
	JrpcRemoveHost       JrpcMethod = "cluster-remove-host"
	JrpcDeactivateDrives JrpcMethod = "cluster-deactivate-drives"
	JrpcDeactivateHosts  JrpcMethod = "cluster-deactivate-hosts"
)

type HostListResponse map[HostId]Host
type DriveListResponse map[DriveId]Drive

type Host struct {
	AddedTime time.Time `json:"added_time"`
	State     string    `json:"state"`
	HostIp    string    `json:"host_ip"`
	Aws       struct {
		InstanceId string `json:"instance_id"`
	} `json:"aws"`
}

type Drive struct {
	HostId         HostId    `json:"host_id"`
	Status         string    `json:"status"`
	Uuid           uuid.UUID `json:"uuid"`
	ShouldBeActive bool      `json:"should_be_active"`
}
