package cleaner

import (
	"github.com/aws/aws-sdk-go/service/sfn"
	"wekactl/internal/aws/scalemachine"
	"wekactl/internal/cluster"
	"wekactl/internal/logging"
)

type ScaleMachine struct {
	StateMachines []*sfn.StateMachineListItem
	ClusterName   cluster.ClusterName
}

func (s *ScaleMachine) Fetch() error {
	stateMachines, err := scalemachine.GetClusterStateMachines(s.ClusterName)
	if err != nil {
		return err
	}
	s.StateMachines = stateMachines
	return nil
}

func (s *ScaleMachine) Delete() error {
	return scalemachine.DeleteStateMachines(s.StateMachines)
}

func (s *ScaleMachine) Print() {
	logging.UserInfo("ScaleMachines:")
	for _, stateMachine := range s.StateMachines {
		logging.UserInfo("\t- %s", *stateMachine.Name)
	}
}
