package iam

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type StatementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

type Principal struct {
	Service string
}

//Resource is prohibited for assume role
type PolicyStatement struct {
	Effect    string
	Action    []string
	Principal Principal
}

type AssumeRolePolicyDocument struct {
	Version   string
	Statement []PolicyStatement
}

func (p PolicyDocument) Bytes() []byte {
	if p.Version == "" {
		p.Version = p.VersionHash()
	}
	policy, err := json.Marshal(&p)
	if err != nil {
		panic(err)
	}
	return policy
}

func (p PolicyDocument) String() string {
	return string(p.Bytes())
}

func (p PolicyDocument) VersionHash() string {
	policy, err := json.Marshal(&p.Statement)
	if err != nil {
		panic(err)
	}
	h := sha256.New()
	h.Write(policy)
	return hex.EncodeToString(h.Sum(nil))
}

func (a AssumeRolePolicyDocument) Bytes() []byte {
	if a.Version == "" {
		a.Version = a.VersionHash()
	}
	policy, err := json.Marshal(&a)
	if err != nil {
		panic(err)
	}
	return policy
}

func (a AssumeRolePolicyDocument) String() string {
	return string(a.Bytes())
}

func (a AssumeRolePolicyDocument) VersionHash() string {
	policy, err := json.Marshal(&a.Statement)
	if err != nil {
		panic(err)
	}
	h := sha256.New()
	h.Write(policy)
	return hex.EncodeToString(h.Sum(nil))
}

func GetLambdaAssumeRolePolicy() AssumeRolePolicyDocument {
	policyDocument := AssumeRolePolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: Principal{
					Service: "lambda.amazonaws.com",
				},
			},
		},
	}

	return policyDocument
}

func GetJoinAndFetchLambdaPolicy() PolicyDocument {
	policyDocument := PolicyDocument{
		Version: "2012-10-17",
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"logs:CreateLogStream",
					"logs:PutLogEvents",
					"logs:CreateLogGroup",
					"dynamodb:GetItem",
					"autoscaling:Describe*",
					"ec2:Describe*",
					"kms:Decrypt",
				},
				Resource: "*",
			},
		},
	}
	return policyDocument
}

func GetStateMachineAssumeRolePolicy() AssumeRolePolicyDocument {
	policyDocument := AssumeRolePolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: Principal{
					Service: "states.amazonaws.com",
				},
			},
		},
	}
	return policyDocument
}

func GetStateMachineRolePolicy() PolicyDocument {
	policyDocument := PolicyDocument{
		Version: "2012-10-17",
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"lambda:InvokeFunction",
				},
				Resource: "*",
			},
		},
	}
	return policyDocument
}

func GetScaleLambdaPolicy() PolicyDocument {
	policyDocument := PolicyDocument{
		Version: "2012-10-17",
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"logs:CreateLogStream",
					"logs:PutLogEvents",
					"logs:CreateLogGroup",
					"ec2:CreateNetworkInterface",
					"ec2:DescribeNetworkInterfaces",
					"ec2:DeleteNetworkInterface",
				},
				Resource: "*",
			},
		},
	}
	return policyDocument
}

func GetTerminateLambdaPolicy() PolicyDocument {
	policyDocument := PolicyDocument{
		Version: "2012-10-17",
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"logs:CreateLogStream",
					"logs:PutLogEvents",
					"logs:CreateLogGroup",
					"ec2:CreateNetworkInterface",
					"ec2:DescribeNetworkInterfaces",
					"ec2:DeleteNetworkInterface",
					"ec2:ModifyInstanceAttribute",
					"autoscaling:Describe*",
					"autoscaling:SetInstanceProtection",
					"ec2:Describe*",
				},
				Resource: "*",
			},
		},
	}
	return policyDocument
}

func GetCloudWatchEventAssumeRolePolicy() AssumeRolePolicyDocument {
	policyDocument := AssumeRolePolicyDocument{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: Principal{
					Service: "events.amazonaws.com",
				},
			},
		},
	}
	return policyDocument
}

func GetCloudWatchEventRolePolicy() PolicyDocument {
	policyDocument := PolicyDocument{
		Version: "2012-10-17",
		Statement: []StatementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"states:StartExecution",
				},
				Resource: "*",
			},
		},
	}
	return policyDocument
}
