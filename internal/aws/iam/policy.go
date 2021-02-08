package iam

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/rs/zerolog/log"
	"wekactl/internal/connectors"
)

func createIamPolicy(policyName string, policy PolicyDocument) (*iam.Policy, error) {
	svc := connectors.GetAWSSession().IAM
	result, err := svc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: aws.String(policy.String()),
		PolicyName:     aws.String(policyName),
	})

	if err != nil {
		fmt.Println("Error", err)
		return nil, err
	}
	log.Debug().Msgf("policy %s was create successfully!", policyName)
	return result.Policy, nil
}
