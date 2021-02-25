package scalemachine

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/rs/zerolog/log"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
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

	stateMachinesOutput, err := svc.ListStateMachines(&sfn.ListStateMachinesInput{})
	if err != nil {
		return
	}
	for _, stateMachine := range stateMachinesOutput.StateMachines {
		if *stateMachine.Name != stateMachineName {
			continue
		}
		tagsOutput, err := svc.ListTagsForResource(&sfn.ListTagsForResourceInput{
			ResourceArn: stateMachine.StateMachineArn,
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
	}

	return
}
