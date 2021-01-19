package scale

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"sort"
	strings2 "strings"
	"sync"
	"time"
	"wekactl/internal/aws/lambdas/protocol"
	"wekactl/internal/connectors"
	"wekactl/internal/lib/jrpc"
	"wekactl/internal/lib/math"
	"wekactl/internal/lib/strings"
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
		c.active = c.ips[0]
		c.clients[c.active] = c.builder(c.active)
		c.Unlock()
	}
	err = c.clients[c.active].Call(c.ctx, string(method), params, result)
	if err != nil {
		if strings2.Contains(err.Error(), "connection refused"){
			c.drop(c.active)
			return c.call(method, params, result)
		}
	}
	return nil
}

type hostState int

func (h hostState) String() string {
	switch h {
	case DEACTIVATING:
		return "DEACTIVATING"
	case HEALTHY:
		return "HEALTHY"
	case UNHEALTHY:
		return "UNHEALTHY"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", h)
	}
}

const (
	/*
		Order matters, it defines priority of hosts removal
	*/
	DEACTIVATING hostState = iota
	UNHEALTHY
	HEALTHY
)

type driveMap map[weka.DriveId]weka.Drive
type nodeMap map[weka.NodeId]weka.Node
type hostInfo struct {
	weka.Host
	id         weka.HostId
	drives     driveMap
	nodes      nodeMap
	scaleState hostState
}

func (host hostInfo) belongsToHg(instances []protocol.HgInstance) bool {
	for _, instance := range instances {
		if host.Aws.InstanceId == instance.Id {
			return true
		}
	}
	return false
}

func (host hostInfo) belongsToHgIpBased(instances []protocol.HgInstance) bool {
	for _, instance := range instances {
		if host.HostIp == instance.PrivateIp {
			return true
		}
	}
	return false
}

func (host hostInfo) numNotHealthyDrives() int {
	notActive := 0
	for _, drive := range host.drives {
		if strings.AnyOf(drive.Status, "INACTIVE", "FAILED") {
			notActive += 1
		}
	}
	return notActive
}

func (host hostInfo) anyDiskBeingRemoved() bool {
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

func (host hostInfo) managementTimedOut() bool {
	for nodeId, node := range host.nodes {
		if !nodeId.IsManagement() {
			continue
		}
		if node.Status == "DOWN" && time.Since(*node.LastFencingTime) > 2*time.Minute {
			return true
		}
	}
	return false
}

func instancesIps(instances []protocol.HgInstance) (ret []string) {
	for _, i := range instances {
		ret = append(ret, i.PrivateIp)
	}
	return
}

func Handler(ctx context.Context, info protocol.HostGroupInfoResponse) (response protocol.ScaleResponse, err error) {
	/*
		Code in here based on following logic:

		A - Fully active, healthy
		T - Desired target number
		U - Unhealthy, we want to remove it for whatever reason. DOWN host, FAILED drive, so on
		D - Drives/hosts being deactivated
		NEW_D - Decision to start deactivating, i.e transition to D, basing on U. Never more then 2 for U

		NEW_D = func(A, U, T, D)

		NEW_D = max(A+U+D-T, min(2-D, U), 0)
	*/
	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, info.Username, info.Password)
	}
	jpool := &jrpcPool{
		ips:     instancesIps(info.Instances),
		clients: map[string]*jrpc.BaseClient{},
		active:  "",
		builder: jrpcBuilder,
		ctx:     ctx,
	}

	systemStatus := weka.StatusResponse{}
	hostsApiList := weka.HostListResponse{}
	driveApiList := weka.DriveListResponse{}
	nodeApiList := weka.NodeListResponse{}

	err = jpool.call(weka.JrpcStatus, struct{}{}, &systemStatus)
	if err != nil {
		return
	}
	err = isAllowedToScale(systemStatus)
	if err != nil {
		return
	}
	err = jpool.call(weka.JrpcHostList, struct{}{}, &hostsApiList)
	if err != nil {
		return
	}
	if info.Role == "backend" {
		err = jpool.call(weka.JrpcDrivesList, struct{}{}, &driveApiList)
		if err != nil {
			return
		}
	}
	err = jpool.call(weka.JrpcNodeList, struct{}{}, &nodeApiList)
	if err != nil {
		return
	}

	hosts := map[weka.HostId]hostInfo{}
	for hostId, host := range hostsApiList {
		hosts[hostId] = hostInfo{
			Host:   host,
			id:     hostId,
			drives: driveMap{},
			nodes: nodeMap{},
		}
	}
	for driveId, drive := range driveApiList {
		if _, ok := hosts[drive.HostId]; ok {
			hosts[drive.HostId].drives[driveId] = drive
		}
	}

	for nodeId, node := range nodeApiList {
		if _, ok := hosts[node.HostId]; ok {
			hosts[node.HostId].nodes[nodeId] = node
		}
	}

	var hostsList []hostInfo
	var inactiveHosts []hostInfo

	for _, host := range hosts {
		switch host.State {
		case "INACTIVE":
			if host.belongsToHgIpBased(info.Instances) {
				inactiveHosts = append(inactiveHosts, host)
			}
		default:
			if !host.belongsToHg(info.Instances) {
				continue
			}
			hostsList = append(hostsList, host)
		}
	}

	calculateHostsState(hostsList)

	sort.Slice(hostsList, func(i, j int) bool {
		// Giving priority to disks to hosts with disk being removed
		// Then hosts with disks not in active state
		// Then hosts sorted by add time
		a := hostsList[i]
		b := hostsList[j]
		if a.scaleState < b.scaleState {
			return true
		}
		if a.scaleState > b.scaleState {
			return false
		}
		if a.numNotHealthyDrives() > b.numNotHealthyDrives() {
			return true
		}
		if a.numNotHealthyDrives() < b.numNotHealthyDrives() {
			return false
		}
		return a.AddedTime.Before(b.AddedTime)
	})

	removeInactive(inactiveHosts, jpool, info.Instances, &response)
	removeOldDrives(driveApiList, jpool, &response)
	numToDeactivate := getNumToDeactivate(hostsList, info.DesiredCapacity)

	deactivateHost := func(host hostInfo) {
		log.Info().Msgf("Trying to deactivate host %s", host.id)
		for _, drive := range host.drives {
			if drive.ShouldBeActive {
				err := jpool.call(weka.JrpcDeactivateDrives, types.JsonDict{
					"drive_uuids": []uuid.UUID{drive.Uuid},
				}, nil)
				if err != nil {
					log.Error().Err(err)
					response.AddTransientError(err, "deactivateDrive")
				}
			}
		}

		if host.allDrivesInactive() {
			jpool.drop(host.HostIp)
			err := jpool.call(weka.JrpcDeactivateHosts, types.JsonDict{
				"host_ids":                 []weka.HostId{host.id},
				"skip_resource_validation": false,
			}, nil)
			if err != nil {
				log.Error().Err(err)
				response.AddTransientError(err, "deactivateHost")
			}
		}

	}

	for _, host := range hostsList[:numToDeactivate] {
		deactivateHost(host)
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

func getNumToDeactivate(hostInfo []hostInfo, desired int) int {
	/*
		A - Fully active, healthy
		T - Target state
		U - Unhealthy, we want to remove it for whatever reason. DOWN host, FAILED drive, so on
		D - Drives/hosts being deactivated
		new_D - Decision to start deactivating, i.e transition to D, basing on U. Never more then 2 for U

		new_D = func(A, U, T, D)

		new_D = max(A+U+D-T, min(2-D, U), 0)
	*/

	nHealthy := 0
	nUnhealthy := 0
	nDeactivating := 0

	for _, host := range hostInfo {
		switch host.scaleState {
		case HEALTHY:
			nHealthy++
		case UNHEALTHY:
			nUnhealthy++
		case DEACTIVATING:
			nDeactivating++
		}
	}

	toDeactivate := calculateDeactivateTarget(nHealthy, nUnhealthy, nDeactivating, desired)
	log.Info().Msgf("%d hosts set to deactivate", toDeactivate)
	return toDeactivate
}

func calculateDeactivateTarget(nHealthy int, nUnhealthy int, nDeactivating int, desired int) int {
	return math.Max(nHealthy+nUnhealthy+nDeactivating-desired, math.Min(2-nDeactivating, nUnhealthy))
}

func isAllowedToScale(status weka.StatusResponse) error {
	if status.IoStatus != "STARTED" {
		return errors.New(fmt.Sprintf("io status:%s, aborting scale", status.IoStatus))
	}

	if status.Upgrade != "" {
		return errors.New("upgrade is running, aborting scale")
	}
	return nil
}

func deriveHostState(host *hostInfo) hostState {
	if host.anyDiskBeingRemoved() {
		return DEACTIVATING
	}
	if strings.AnyOf(host.State, "DEACTIVATING", "REMOVING", "INACTIVE") {
		return DEACTIVATING
	}
	if host.Status == "DOWN" && host.managementTimedOut() {
		return UNHEALTHY
	}
	if host.numNotHealthyDrives() > 0 {
		return UNHEALTHY
	}
	return HEALTHY
}

func calculateHostsState(hosts []hostInfo) {
	for i := range hosts {
		host := &hosts[i]
		host.scaleState = deriveHostState(host)
	}
}

func selectInstanceByIp(ip string, instances []protocol.HgInstance) *protocol.HgInstance {
	for _, i := range instances {
		if i.PrivateIp == ip {
			return &i
		}
	}
	return nil
}

func removeInactive(hosts []hostInfo, jpool *jrpcPool, instances []protocol.HgInstance, p *protocol.ScaleResponse) {
	for _, host := range hosts {
		jpool.drop(host.HostIp)
		err := jpool.call(weka.JrpcRemoveHost, types.JsonDict{
			"host_id": host.id.Int(),
		}, nil)
		if err != nil {
			log.Error().Err(err)
			p.AddTransientError(err, "removeInactive")
			continue
		}
		instance := selectInstanceByIp(host.HostIp, instances)
		if instance != nil {
			p.ToTerminate = append(p.ToTerminate, *instance)
		}

		for _, drive := range host.drives {
			removeDrive(jpool, drive, p)
		}
	}
	return
}

func removeOldDrives(drives weka.DriveListResponse, jpool *jrpcPool, p *protocol.ScaleResponse) {
	for _, drive := range drives {
		if drive.HostId.Int() == -1 && drive.Status == "INACTIVE" {
			removeDrive(jpool, drive, p)
		}
	}
}

func removeDrive(jpool *jrpcPool, drive weka.Drive, p *protocol.ScaleResponse) {
	err := jpool.call(weka.JrpcRemoveDrive, types.JsonDict{
		"drive_uuids": []uuid.UUID{drive.Uuid},
	}, nil)
	if err != nil {
		log.Error().Err(err)
		p.AddTransientError(err, "removeDrive")
	}
}
