package iam

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/rs/zerolog/log"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/connectors"
)

func getIAMTags(hostGroupInfo hostgroups.HostGroupInfo) []*iam.Tag {
	var iamTags []*iam.Tag
	for key, value := range common.GetHostGroupTags(hostGroupInfo) {
		iamTags = append(iamTags, &iam.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return iamTags
}
func CreateIamRole(hostGroupInfo hostgroups.HostGroupInfo, roleName, policyName string, assumeRolePolicy AssumeRolePolicyDocument, policy PolicyDocument) (*string, error) {
	log.Debug().Msgf("creating role %s", roleName)
	svc := connectors.GetAWSSession().IAM
	input := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicy.String()),
		Path:                     aws.String("/"),
		//max roleName length must be 64 characters
		RoleName: aws.String(roleName),
		Tags:     getIAMTags(hostGroupInfo),
	}

	result, err := svc.CreateRole(input)
	if err != nil {
		return nil, err
	}

	err = svc.WaitUntilRoleExists(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("role %s was created successfully!", roleName)

	if policy.Version != "" {
		policyOutput, err := createIamPolicy(policyName, policy)
		if err != nil {
			return nil, err
		}

		_, err = svc.AttachRolePolicy(&iam.AttachRolePolicyInput{PolicyArn: policyOutput.Arn, RoleName: &roleName})
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("policy %s was attached successfully!", policyName)
	}

	return result.Role.Arn, nil
}
