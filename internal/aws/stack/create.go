package stack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"time"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
)

type Cluster struct {
	Count        int    `json:"count"`
	InstanceType string `json:"instance_type"`
	Role         string `json:"role"`
}

type Response struct {
	Url              string            `json:"url"`
	QuickCreateStack map[string]string `json:"quick_create_stack"`
}

func CreateStack(createParams cluster.CreateParams) error {
	log.Debug().Msgf("fetching template url ...")
	templateUrl, err := getTemplateUrl(createParams.Count, createParams.InstanceType, createParams.WekaVersion, createParams.Token)
	if err != nil {
		return err
	}
	log.Debug().Msgf("creating stack using template url: %s ...", templateUrl)
	svc := connectors.GetAWSSession().CF
	_, err = svc.CreateStack(&cloudformation.CreateStackInput{
		StackName:   &createParams.Name,
		TemplateURL: &templateUrl,
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("VpcId"),
				ParameterValue: &createParams.VpcId,
			},
			{
				ParameterKey:   aws.String("SubnetId"),
				ParameterValue: &createParams.SubnetId,
			},
			{
				ParameterKey:   aws.String("NetworkTopology"),
				ParameterValue: &createParams.NetworkTopology,
			},
			{
				ParameterKey:   aws.String("CustomProxy"),
				ParameterValue: &createParams.CustomProxy,
			},
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: &createParams.KeyName,
			},
			{
				ParameterKey:   aws.String("DistToken"),
				ParameterValue: &createParams.Token,
			},
		},
		Capabilities: []*string{
			aws.String("CAPABILITY_NAMED_IAM"),
		},
	})
	return err
}

func getTemplateUrl(count int, instanceType, wekaVersion, token string) (templateUrl string, err error) {
	env.Config.Region = "eu-central-1"

	baseUrl := "https://get.weka.io"
	requestUrl := fmt.Sprintf("%s/dist/v1/aws/cfn/%s", baseUrl, wekaVersion)

	payload := map[string][]Cluster{"cluster": {
		Cluster{
			Count:        count,
			InstanceType: instanceType,
			Role:         "backend",
		},
	},
	}

	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonValue))
	if err != nil {
		return
	}

	req.SetBasicAuth(token, "")
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var res Response
	err = json.Unmarshal(body, &res)
	if err != nil {
		return
	}
	templateUrl = res.Url
	return
}

func WaitForStackCreationComplete(stackName string) (status string, err error) {
	svc := connectors.GetAWSSession().CF
	var res *cloudformation.DescribeStacksOutput

	for {
		res, err = svc.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: &stackName,
		})
		if err != nil {
			return
		}
		status = *res.Stacks[0].StackStatus

		if status != cloudformation.StackStatusCreateInProgress {
			break
		}

		log.Debug().Msgf("Stack isn't ready yet, going to sleep for 1M ...")
		time.Sleep(time.Minute)
	}

	return
}
