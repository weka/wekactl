package lambdas

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/lithammer/dedent"
	"math/rand"
	"strings"
	"time"
	"wekactl/internal/aws/common"
	"wekactl/internal/connectors"
)

type BackendCoreCount struct {
	total    int
	frontend int
	drive    int
}

type BackendCoreCounts map[string]BackendCoreCount

func shuffleSlice(slice []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func getBackendCoreCounts() BackendCoreCounts {
	backendCoreCounts := BackendCoreCounts{
		"r3.large":      BackendCoreCount{total: 1, frontend: 0, drive: 0},
		"r3.xlarge":     BackendCoreCount{total: 1, frontend: 0, drive: 0},
		"r3.2xlarge":    BackendCoreCount{total: 3, frontend: 1, drive: 1},
		"r3.4xlarge":    BackendCoreCount{total: 7, frontend: 1, drive: 1},
		"r3.8xlarge":    BackendCoreCount{total: 7, frontend: 1, drive: 2},
		"i3.large":      BackendCoreCount{total: 1, frontend: 0, drive: 0},
		"i3.xlarge":     BackendCoreCount{total: 1, frontend: 0, drive: 0},
		"i3.2xlarge":    BackendCoreCount{total: 3, frontend: 1, drive: 1},
		"i3.4xlarge":    BackendCoreCount{total: 7, frontend: 1, drive: 1},
		"i3.8xlarge":    BackendCoreCount{total: 7, frontend: 1, drive: 2},
		"i3.16xlarge":   BackendCoreCount{total: 14, frontend: 1, drive: 4},
		"i3en.large":    BackendCoreCount{total: 1, frontend: 0, drive: 0},
		"i3en.xlarge":   BackendCoreCount{total: 1, frontend: 0, drive: 0},
		"i3en.2xlarge":  BackendCoreCount{total: 3, frontend: 1, drive: 1},
		"i3en.3xlarge":  BackendCoreCount{total: 3, frontend: 1, drive: 1},
		"i3en.6xlarge":  BackendCoreCount{total: 7, frontend: 1, drive: 2},
		"i3en.12xlarge": BackendCoreCount{total: 7, frontend: 1, drive: 2},
		"i3en.24xlarge": BackendCoreCount{total: 14, frontend: 1, drive: 4},
	}
	return backendCoreCounts
}

func GetJoinParams(asgName, tableName, role string) (string, error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return "", err
	}

	instanceIds := common.GetInstanceIdsFromAutoScalingGroupOutput(asgOutput)
	instances, err := common.GetInstances(instanceIds)
	if err != nil {
		return "", err
	}
	ips := common.GetInstancesIps(instances)
	instanceType := common.GetInstanceTypeFromAutoScalingGroupOutput(asgOutput)
	shuffleSlice(ips)
	creds, err := getUsernameAndPassword(tableName)
	if err != nil {
		return "", err
	}

	bashScriptTemplate := `
	#!/bin/bash

	set -ex

	export WEKA_USERNAME="%s"
	export WEKA_PASSWORD="%s"
	export WEKA_RUN_CREDS="-e WEKA_USERNAME=$WEKA_USERNAME -e WEKA_PASSWORD=$WEKA_PASSWORD"
	declare -a backend_ips=("%s" )

	random=$$
	echo $random
	for backend_ip in ${backend_ips[@]}; do
		VERSION=$(curl -s -XPOST --data '{"jsonrpc":"2.0", "method":"client_query_backend", "id":"'$random'"}' $backend_ip:14000/api/v1 | sed  's/.*"software_release":"\([^"]*\)".*$/\1/g')
		if [[ "$VERSION" != "" ]]; then
			break
		fi
	done

	curl $backend_ip:14000/dist/v1/install | sh

	weka version get --from $backend_ip:14000 $VERSION --set-current
	weka version prepare $VERSION
	weka local stop && weka local rm --all -f
	weka local setup host --cores %d --frontend-dedicated-cores %d --drives-dedicated-cores %d --join-ips %s`

	isReady := `
	while ! weka debug manhole -s 0 operational_status | grep '"is_ready": true' ; do
		sleep 1
	done
	echo Connected to cluster
	`

	addDrives := `
	host_id=$(weka local run $WEKA_RUN_CREDS manhole getServerInfo | grep hostIdValue: | awk '{print $2}')
	mkdir -p /opt/weka/tmp
	cat >/opt/weka/tmp/find_drives.py <<EOL
	import json
	import sys
	for d in json.load(sys.stdin)['disks']:
		if d['isRotational']: continue
		if d['type'] != 'DISK': continue
		if d['isMounted']: continue
		if d['model'] != 'Amazon EC2 NVMe Instance Storage': continue
		print(d['devPath'])
	EOL
	devices=$(weka local run $WEKA_RUN_CREDS bash -ce 'wapi machine-query-info --info-types=DISKS -J | python3 /opt/weka/tmp/find_drives.py')
	for device in $devices; do
		weka cluster drive add $host_id $device
	done
	`
	var cores, frontend, drive int
	if role == "backend" {
		backendCoreCounts := getBackendCoreCounts()
		cores = backendCoreCounts[instanceType].total
		frontend = backendCoreCounts[instanceType].frontend
		drive = backendCoreCounts[instanceType].drive
		bashScriptTemplate += " --dedicate" + isReady + addDrives
	} else {
		bashScriptTemplate += isReady
		cores = 1
		frontend = 1
		drive = 0
	}

	bashScript := fmt.Sprintf(
		dedent.Dedent(bashScriptTemplate),
		creds.Username,
		creds.Password,
		strings.Join(ips, "\" \""),
		cores,
		frontend,
		drive,
		strings.Join(ips, ","),
	)

	return bashScript, nil
}
