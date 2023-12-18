package lambdas

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/lithammer/dedent"
	"github.com/weka/go-cloud-lib/bash_functions"
	common2 "github.com/weka/go-cloud-lib/common"
	"github.com/weka/go-cloud-lib/functions_def"
	"github.com/weka/go-cloud-lib/join"
	"wekactl/internal/aws/common"
	"wekactl/internal/connectors"
)

type AwsFuncDef struct {
}

func (d *AwsFuncDef) GetFunctionCmdDefinition(name functions_def.FunctionName) string {
	defTemplate := `
	function %s {
		echo "currently ${FUNCNAME[0]} is not supported, ignoring..."
	}
	`
	return dedent.Dedent(fmt.Sprintf(defTemplate, name))
}

func GetJoinParams(ctx context.Context, clusterName, asgName, tableName, role string) (string, error) {
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
	common2.ShuffleSlice(ips)
	creds, err := GetUsernameAndPassword(tableName)
	if err != nil {
		return "", err
	}

	scriptBase := `
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
	`

	findDrivesScript := `
	import json
	import sys
	for d in json.load(sys.stdin)['disks']:
		if d['isRotational']: continue
		if d['type'] != 'DISK': continue
		if d['isMounted']: continue
		if d['model'] != 'Amazon EC2 NVMe Instance Storage': continue
		print(d['devPath'])
	`

	backendCoreCounts := common.GetBackendCoreCounts()
	instanceParams := backendCoreCounts[instanceType]

	joinParams := join.JoinParams{
		WekaUsername:   "admin",
		WekaPassword:   creds.Password,
		IPs:            ips,
		InstallDpdk:    true,
		InstanceParams: instanceParams,
	}

	joinScriptGenerator := join.JoinScriptGenerator{
		FailureDomainCmd:   bash_functions.GetHashedPrivateIpBashCmd(),
		GetInstanceNameCmd: "",
		FindDrivesScript:   dedent.Dedent(findDrivesScript),
		ScriptBase:         dedent.Dedent(scriptBase),
		Params:             joinParams,
		FuncDef:            &AwsFuncDef{},
	}
	bashScript := joinScriptGenerator.GetJoinScript(ctx)

	return bashScript, nil
}
