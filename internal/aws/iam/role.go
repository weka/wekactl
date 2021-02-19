package iam

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/rs/zerolog/log"
	"strings"
	"wekactl/internal/aws/common"
	"wekactl/internal/aws/hostgroups"
	"wekactl/internal/connectors"
)

func GetIAMTags(hostGroupInfo hostgroups.HostGroupInfo, version string) []*iam.Tag {
	var iamTags []*iam.Tag
	for key, value := range common.GetHostGroupTags(hostGroupInfo, version) {
		iamTags = append(iamTags, &iam.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return iamTags
}
func CreateIamRole(tags []*iam.Tag, roleName, policyName string, assumeRolePolicy AssumeRolePolicyDocument, policy PolicyDocument) (*string, error) {
	log.Debug().Msgf("creating role %s", roleName)
	svc := connectors.GetAWSSession().IAM
	input := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicy.String()),
		Path:                     aws.String("/"),
		//max roleName length must be 64 characters
		RoleName: aws.String(roleName),
		Tags:     tags,
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

func getIamRole(roleBaseName string, marker *string) (iamRole *iam.Role, err error) {
	svc := connectors.GetAWSSession().IAM

	rolesOutput, err := svc.ListRoles(&iam.ListRolesInput{Marker: marker})
	if err != nil {
		return
	}
	for _, role := range rolesOutput.Roles {
		if strings.Contains(*role.RoleName, roleBaseName) {
			return role, nil
		}
	}

	if *rolesOutput.IsTruncated {
		return getIamRole(roleBaseName, rolesOutput.Marker)
	}

	return
}

func deleteLeftoverPolicies(policyName string, marker *string) error {
	// Handling a case that a policy exists although it isn't attached to a role
	svc := connectors.GetAWSSession().IAM
	policiesOutput, err := svc.ListPolicies(&iam.ListPoliciesInput{Marker: marker})
	if err != nil {
		return err
	}
	for _, policy := range policiesOutput.Policies {
		if *policy.PolicyName == policyName {
			err = deleteIamPolicy(policy.Arn)
			if err != nil {
				return err
			}
			log.Debug().Msgf("leftover policy %s was deleted successfully", *policy.PolicyName)
		}
	}

	if *policiesOutput.IsTruncated {
		return deleteLeftoverPolicies(policyName, policiesOutput.Marker)
	}

	return nil
}

func DeleteIamRole(roleBaseName, policyName string) error {
	svc := connectors.GetAWSSession().IAM
	role, err := getIamRole(roleBaseName, nil)
	if err != nil {
		return err
	}
	if role == nil {
		return nil
	}

	result, err := svc.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: role.RoleName,
	})
	if err != nil {
		return err
	}

	for _, policy := range result.AttachedPolicies {
		_, err = svc.DetachRolePolicy(&iam.DetachRolePolicyInput{
			RoleName:  role.RoleName,
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("policy %s detached", *policy.PolicyName)

		err = deleteIamPolicy(policy.PolicyArn)
		if err != nil {
			return err
		}
		log.Debug().Msgf("policy %s was deleted successfully", *policy.PolicyName)
	}

	_, err = svc.DeleteRole(&iam.DeleteRoleInput{RoleName: role.RoleName})
	if err != nil {
		return err
	}
	log.Debug().Msgf("role %s was deleted successfully", *role.RoleName)

	return deleteLeftoverPolicies(policyName, nil)
}
