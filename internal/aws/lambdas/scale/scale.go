package scale

import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"sort"
	"sync"
	"wekactl/internal/aws/lambdas/protocol"
	"wekactl/internal/connectors"
	"wekactl/internal/lib/jrpc"
	"wekactl/internal/lib/types"
	"wekactl/internal/lib/weka"
)

type jrpcClientBuilder func(ip string) *jrpc.BaseClient
type jrpcPool struct {
	sync.RWMutex
	ips     []string
	clients map[string]*jrpc.BaseClient
	active  string
	builder jrpcClientBuilder
	ctx     context.Context
}

func (c *jrpcPool) drop(toDrop string) {
	c.Lock()
	defer c.Unlock()
	if c.active == toDrop {
		c.active = ""
	}

	for i, ip := range c.ips {
		if ip == toDrop {
			c.ips[i] = c.ips[len(c.ips)-1]
			c.ips = c.ips[:len(c.ips)-1]
			break
		}
	}
}

func (c *jrpcPool) call(method weka.JrpcMethod, params, result interface{}) (err error) {
	if c.active == "" {
		c.Lock()
		c.active = c.ips[len(c.ips)-1]
		c.clients[c.active] = c.builder(c.active)
		c.Unlock()
	}
	return c.clients[c.active].Call(c.ctx, string(method), params, result)
}

type driveMap map[weka.DriveId]weka.Drive
type hostInfo struct {
	weka.Host
	id     weka.HostId
	drives driveMap
}

func (host hostInfo) belongsToHg(instanceIds []string) bool {
	for _, instanceId := range instanceIds {
		if host.Aws.InstanceId == instanceId {
			return true
		}
	}
	return false
}

func (host hostInfo) numNotActiveDrives() int {
	notActive := 0
	for _, drive := range host.drives {
		if drive.Status != "ACTIVE" && drive.Status != "PHASING_IN" {
			notActive += 1
		}
	}
	return notActive
}

func (host hostInfo) areDiskBeingRemoved() bool {
	for _, drive := range host.drives {
		if !drive.ShouldBeActive {
			return true
		}
	}
	return false
}

func (host hostInfo) allDrivesInactive() bool {
	for _, drive := range host.drives {
		if drive.Status != "INACTIVE" {
			return false
		}
	}
	return true
}

func Handler(ctx context.Context, info protocol.HostGroupInfoResponse) (response protocol.ScaleResponse, err error) {
	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, info.Username, info.Password)
	}
	jpool := &jrpcPool{
		ips:     info.PrivateIps,
		clients: map[string]*jrpc.BaseClient{},
		active:  "",
		builder: jrpcBuilder,
		ctx:     ctx,
	}

	hostsApiList := weka.HostListResponse{}
	driveApiList := weka.DriveListResponse{}

	err = jpool.call(weka.JrpcHostList, struct{}{}, &hostsApiList)
	if err != nil {
		return
	}
	err = jpool.call(weka.JrpcDrivesList, struct{}{}, &driveApiList)
	if err != nil {
		return
	}

	hosts := map[weka.HostId]hostInfo{}
	for hostId, host := range hostsApiList {
		hosts[hostId] = hostInfo{
			Host:   host,
			id:     hostId,
			drives: driveMap{},
		}
	}
	for driveId, drive := range driveApiList {
		if _, ok := hosts[drive.HostId]; ok {
			hosts[drive.HostId].drives[driveId] = drive
		}
	}

	var hostsList []hostInfo
	var inactiveHosts []hostInfo

	for _, host := range hosts {
		switch host.State {
		case "INACTIVE":
			// TODO: FetchInfo can return structs of instances, with instance id + private ip
			// Then we can determine if inactive host belongs to HG basing on IP as well
			inactiveHosts = append(inactiveHosts, host)
		default:
			if !host.belongsToHg(info.InstanceIds) {
				continue
			}
			hostsList = append(hostsList, host)
		}
	}

	sort.Slice(hostsList, func(i, j int) bool {
		// Giving priority to disks to hosts with disk being removed
		// Then hosts with disks not in active state
		// Then hosts sorted by add time
		a := hostsList[i]
		b := hostsList[j]
		if !a.areDiskBeingRemoved() && b.areDiskBeingRemoved() {
			return true
		}
		if a.numNotActiveDrives() < b.numNotActiveDrives() {
			return true
		}
		return a.AddedTime.Before(b.AddedTime)
	})

	response.AddTransientErrors(removeInactive(inactiveHosts, jpool), "removeInactive")
	numToDeactivate := len(hostsList) - info.DesiredCapacity

	if numToDeactivate > 0 {
		response.AddTransientErrors(deactivateDrives(hostsList[0:numToDeactivate], jpool), "deactivateDrives")
		response.AddTransientErrors(deactivateHosts(hostsList[0:numToDeactivate], jpool), "deactivateHosts")
	}

	for _, host := range hostsList {
		response.Hosts = append(response.Hosts, protocol.ScaleResponseHost{
			InstanceId: host.Aws.InstanceId,
			State:      host.State,
			AddedTime:  host.AddedTime,
			HostId:     host.id,
		})
	}
	return
}

func deactivateHosts(hosts []hostInfo, jpool *jrpcPool) (errs []error) {
	for _, host := range hosts {
		if host.allDrivesInactive() {
			jpool.drop(host.HostIp)
			err := jpool.call(weka.JrpcDeactivateHosts, types.JsonDict{
				"host_ids":                 []weka.HostId{host.id},
				"skip_resource_validation": false,
			}, nil)
			if err != nil {
				log.Error().Err(err)
				errs = append(errs, err)
			}
		}
	}
	return
}

func deactivateDrives(hosts []hostInfo, jpool *jrpcPool) (errs []error) {
	for _, host := range hosts {
		for _, drive := range host.drives {
			if drive.ShouldBeActive {
				err := jpool.call(weka.JrpcDeactivateDrives, types.JsonDict{
					"drive-uuids": []uuid.UUID{drive.Uuid},
				}, nil)
				if err != nil {
					log.Error().Err(err)
					errs = append(errs, err)
				}
			}
		}
	}
	return
}

func removeInactive(hosts []hostInfo, jpool *jrpcPool) (errors []error) {
	for _, host := range hosts {
		jpool.drop(host.HostIp)
		err := jpool.call(weka.JrpcRemoveHost, types.JsonDict{
			"host_id": host.id.Int(),
		}, nil)
		if err != nil {
			log.Error().Err(err)
			errors = append(errors, err)
		}
	}
	return
}
