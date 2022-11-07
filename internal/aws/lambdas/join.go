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

func shuffleSlice(slice []string) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) { slice[i], slice[j] = slice[j], slice[i] })
}

func GetJoinParams(clusterName, asgName, tableName, role string) (string, error) {
	svc := connectors.GetAWSSession().ASG
	input := &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{&asgName}}
	asgOutput, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		return "", err
	}

	ips, err := common.GetBackendsPrivateIps(clusterName)
	if err != nil {
		return "", err
	}
	instanceType := common.GetInstanceTypeFromAutoScalingGroupOutput(asgOutput)
	shuffleSlice(ips)
	creds, err := getUsernameAndPassword(tableName)
	if err != nil {
		return "", err
	}

	bashScriptTemplate := `
	#!/bin/bash

	set -ex

	function setup_aws_logs_agent() {
		echo "---------------------------"
		echo " Setting up AWS logs agent "
		echo "---------------------------"

		no_proxy=".amazonaws.com" https_proxy="${PROXY}" retry 5 3 yum install -y amazon-cloudwatch-agent.x86_64 || return 1
		configure_aws_logs_agent || return 1
		service amazon-cloudwatch-agent restart || return 1
	}

	function create_wekaio_partition() {
		echo "--------------------------------------------"
		echo " Creating local filesystem on WekaIO volume "
		echo "--------------------------------------------"

		if [ -e /dev/xvdp ]
		then
			wekaiosw_device="/dev/xvdp"
		elif [ -e /dev/sdp ]
		then
			wekaiosw_device="/dev/sdp"
		elif [ -e /dev/nvme1n1 ]
		then
			wekaiosw_device="/dev/nvme1n1"
		else
			echo "error: Could not find the WekaIO software block device. This may be the result of a new kernel not exposing the device in the known paths."
			return 1
		fi

		sleep 4
		mkfs.ext4 -L wekaiosw ${wekaiosw_device} || return 1
		mkdir -p /opt/weka || return 1
		mount $wekaiosw_device /opt/weka || return 1
		echo "LABEL=wekaiosw /opt/weka ext4 defaults 0 2" >>/etc/fstab
	}

	setup_aws_logs_agent || echo "setup_aws_logs_agent failed" >> /tmp/res
	df -h > /tmp/df_res
	create_wekaio_partition || echo "create_wekaio_partition failed" >> /tmp/res

	export WEKA_USERNAME="%s"
	export WEKA_PASSWORD="%s"
	export WEKA_RUN_CREDS="-e WEKA_USERNAME=$WEKA_USERNAME -e WEKA_PASSWORD=$WEKA_PASSWORD"
	declare -a backend_ips=("%s" )

	random=$$
	echo $random
	for backend_ip in ${backend_ips[@]}; do
		if VERSION=$(curl -s -XPOST --data '{"jsonrpc":"2.0", "method":"client_query_backend", "id":"'$random'"}' $backend_ip:14000/api/v1 | sed  's/.*"software_release":"\([^"]*\)".*$/\1/g'); then
			if [[ "$VERSION" != "" ]]; then
				break
			fi
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
		backendCoreCounts := common.GetBackendCoreCounts()
		instanceParams := backendCoreCounts[instanceType]
		cores = instanceParams.Total
		frontend = instanceParams.Frontend
		drive = instanceParams.Drive
		if !instanceParams.Converged {
			bashScriptTemplate += " --dedicate"
		}
		bashScriptTemplate += isReady + addDrives
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
