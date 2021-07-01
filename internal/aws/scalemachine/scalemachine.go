package scalemachine

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
	"wekactl/internal/env"
)

func CreateStateMachine(tags []*sfn.Tag, lambda StateMachineLambdasArn, roleArn, stateMachineName string) (*string, error) {
	svc := connectors.GetAWSSession().SFN

	states := make(map[string]interface{})
	states["HostGroupInfo"] = NextState{
		Type:     "Task",
		Resource: lambda.Fetch,
		Next:     "Scale",
	}
	states["Scale"] = NextState{
		Type:     "Task",
		Resource: lambda.Scale,
		Next:     "Terminate",
	}
	states["Terminate"] = NextState{
		Type:     "Task",
		Resource: lambda.Terminate,
		Next:     "ErrorCheck",
	}

	states["ErrorCheck"] = IsNullChoiceState{
		Type: "Choice",
		Choices: []IsNullChoice{
			{
				Variable: "$.TransientErrors",
				IsNull:   false,
				Next:     "Transient",
			},
		},
		Default: "Success",
	}

	states["Success"] = SuccessState{
		Type: "Succeed",
	}

	states["Transient"] = EndState{
		Type:     "Task",
		Resource: lambda.Transient,
		End:      true,
	}
	stateMachine := StateMachine{
		Comment: "Wekactl state machine",
		StartAt: "HostGroupInfo",
		States:  states,
	}

	b, err := json.Marshal(&stateMachine)
	if err != nil {
		log.Debug().Msg("Error marshaling stateMachine")
		return nil, err
	}
	definition := string(b)
	log.Debug().Msgf("Creating state machine :%s", stateMachineName)

	result, err := svc.CreateStateMachine(&sfn.CreateStateMachineInput{
		Name:       aws.String(stateMachineName),
		RoleArn:    &roleArn,
		Tags:       tags,
		Definition: aws.String(definition),
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("State machine %s was created successfully!", stateMachineName)
	return result.StateMachineArn, nil
}

func DeleteStateMachine(stateMachineName string) error {
	svc := connectors.GetAWSSession().SFN

	stateMachinesOutput, err := svc.ListStateMachines(&sfn.ListStateMachinesInput{})
	if err != nil {
		return err
	}
	for _, stateMachine := range stateMachinesOutput.StateMachines {
		if *stateMachine.Name != stateMachineName {
			continue
		}
		_, err = svc.DeleteStateMachine(&sfn.DeleteStateMachineInput{
			StateMachineArn: stateMachine.StateMachineArn,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("state machine %s was deleted successfully", stateMachineName)
	}

	return nil
}

func GetStateMachineVersion(stateMachineName string) (version string, err error) {
	svc := connectors.GetAWSSession().SFN
	stateMachineArn, err := GetStateMachineArn(stateMachineName)
	if err != nil {
		return
	}

	tagsOutput, err := svc.ListTagsForResource(&sfn.ListTagsForResourceInput{
		ResourceArn: &stateMachineArn,
	})
	if err != nil {
		return "", err
	}
	for _, tag := range tagsOutput.Tags {
		if *tag.Key == cluster.VersionTagKey {
			version = *tag.Value
			return version, nil
		}
	}

	return
}

func GetStateMachineArn(stateMachineName string) (arn string, err error) {
	account, err := common.GetAccountId()
	if err != nil {
		return
	}
	arn = fmt.Sprintf("arn:aws:states:%s:%s:stateMachine:%s", env.Config.Region, account, stateMachineName)
	return
}

func GetClusterStateMachines(clusterName cluster.ClusterName) (stateMachines []*sfn.StateMachineListItem, err error) {
	svc := connectors.GetAWSSession().SFN
	stateMachinesOutput, err := svc.ListStateMachines(&sfn.ListStateMachinesInput{})
	if err != nil {
		return
	}

	var tagsOutput *sfn.ListTagsForResourceOutput

	for _, stateMachine := range stateMachinesOutput.StateMachines {
		tagsOutput, err = svc.ListTagsForResource(&sfn.ListTagsForResourceInput{
			ResourceArn: stateMachine.StateMachineArn,
		})
		if err != nil {
			return
		}
		for _, tag := range tagsOutput.Tags {
			if *tag.Key == cluster.ClusterNameTagKey && *tag.Value == string(clusterName) {
				stateMachines = append(stateMachines, stateMachine)
				break
			}
		}
	}
	return

}

func DeleteStateMachines(stateMachines []*sfn.StateMachineListItem) error {
	for _, stateMachine := range stateMachines {
		err := DeleteStateMachine(*stateMachine.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetStateMachineRoleArn(stateMachineName string) (arn string, err error) {
	stateMachineArn, err := GetStateMachineArn(stateMachineName)
	if err != nil {
		return
	}

	svc := connectors.GetAWSSession().SFN
	stateMachineOutput, err := svc.DescribeStateMachine(&sfn.DescribeStateMachineInput{StateMachineArn: &stateMachineArn})
	if err != nil {
		return
	}
	arn = *stateMachineOutput.RoleArn

	return
}

func UpdateStateMachineRoleArn(stateMachineArn, roleArn string) error {
	svc := connectors.GetAWSSession().SFN
	_, err := svc.UpdateStateMachine(&sfn.UpdateStateMachineInput{
		StateMachineArn: &stateMachineArn,
		RoleArn:         &roleArn,
	})
	return err
}
