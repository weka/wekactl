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

func deleteIamPolicy(policyArn *string) error {
	svc := connectors.GetAWSSession().IAM
	_, err := svc.DeletePolicy(&iam.DeletePolicyInput{
		PolicyArn: policyArn,
	})

	return err
}

func GetRolesPolicies(roles []*iam.Role) (rolePolicies map[string][]*iam.AttachedPolicy, err error) {
	var policiesOutput *iam.ListAttachedRolePoliciesOutput
	rolePolicies = make(map[string][]*iam.AttachedPolicy)
	svc := connectors.GetAWSSession().IAM
	log.Debug().Msg("fetching cluster policies ...")
	for _, role := range roles {
		policiesOutput, err = svc.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
			RoleName: role.RoleName,
		})
		if err != nil {
			return
		}
		rolePolicies[*role.RoleName] = policiesOutput.AttachedPolicies
	}
	return
}
