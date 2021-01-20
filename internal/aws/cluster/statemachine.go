package cluster


type ScaleMachine struct {
	ResourceName string
	HGInfo Lambda
	Scale Lambda
	Terminate Lambda
	StateMachine StateMachine
}


type NextState struct {
	Type     string
	Resource string
	Next     string
}

type EndState struct {
	Type     string
	Resource string
	End      bool
}

type SuccessState struct {
	Type string
}

type StateMachine struct {
	Comment string
	StartAt string
	States  map[string]interface{}
}

type StateMachineLambdas struct {
	Fetch     string
	Scale     string
	Terminate string
	Transient string
}


type IsNullChoice struct {
	Variable string
	IsNull   bool
	Next     string
}

type IsNullChoiceState struct {
	Type    string
	Choices []IsNullChoice
	Default string
}

