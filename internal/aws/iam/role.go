package iam

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
	"strings"
	"sync"
	"wekactl/internal/cluster"
	"wekactl/internal/connectors"
)

func attachIamPolicy(roleName, policyName string, policy PolicyDocument) error {
	svc := connectors.GetAWSSession().IAM

	_, err := svc.PutRolePolicy(&iam.PutRolePolicyInput{
		PolicyDocument: aws.String(policy.String()),
		PolicyName:     &policyName,
		RoleName:       &roleName,
	})
	if err != nil {
		return err
	}
	log.Debug().Msgf("policy %s was attached successfully!", policyName)
	return nil
}

func CreateIamRole(clusterName cluster.ClusterName, tags []*iam.Tag, roleName, policyName string, assumeRolePolicy AssumeRolePolicyDocument, policy PolicyDocument) (*string, error) {
	log.Debug().Msgf("creating role %s", roleName)
	svc := connectors.GetAWSSession().IAM
	input := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicy.String()),
		Path:                     aws.String(fmt.Sprintf("/wekactl/%s/", clusterName)),
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
		err = attachIamPolicy(roleName, policyName, policy)
		if err != nil {
			return nil, err
		}
	}

	return result.Role.Arn, nil
}

func getIamRole(clusterName cluster.ClusterName, roleBaseName string, marker *string) (iamRole *iam.Role, err error) {
	svc := connectors.GetAWSSession().IAM

	rolesOutput, err := svc.ListRoles(&iam.ListRolesInput{
		Marker:     marker,
		PathPrefix: aws.String(fmt.Sprintf("/wekactl/%s/", clusterName)),
	})
	if err != nil {
		return
	}
	for _, role := range rolesOutput.Roles {
		if strings.Contains(*role.RoleName, roleBaseName) {
			return role, nil
		}
	}

	if *rolesOutput.IsTruncated {
		return getIamRole(clusterName, roleBaseName, rolesOutput.Marker)
	}

	return
}

func removeRolePolicy(role *iam.Role) error {
	log.Debug().Msgf("listing role %s policies", *role.RoleName)
	svc := connectors.GetAWSSession().IAM
	result, err := svc.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: role.RoleName,
	})
	if err != nil {
		return err
	}

	for _, policyName := range result.PolicyNames {
		log.Debug().Msgf("deleting role %s policy %s", *role.RoleName, *policyName)
		_, err = svc.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			RoleName:   role.RoleName,
			PolicyName: policyName,
		})
		if err != nil {
			return err
		}
		log.Debug().Msgf("policy %s was deleted successfully", *policyName)
	}
	return nil
}

func DeleteIamRole(clusterName cluster.ClusterName, roleBaseName string) error {
	svc := connectors.GetAWSSession().IAM
	log.Debug().Msgf("fetching role %s ...", roleBaseName)
	role, err := getIamRole(clusterName, roleBaseName, nil)
	if err != nil {
		return err
	}
	if role == nil {
		return nil
	}

	err = removeRolePolicy(role)
	if err != nil {
		return err
	}

	log.Debug().Msgf("deleting role %s", *role.RoleName)
	_, err = svc.DeleteRole(&iam.DeleteRoleInput{RoleName: role.RoleName})
	if err != nil {
		return err
	}
	log.Debug().Msgf("role %s was deleted successfully", *role.RoleName)
	return nil

}

func GetIamRoleName(clusterName cluster.ClusterName, roleBaseName string) (roleName string, err error) {
	role, err := getIamRole(clusterName, roleBaseName, nil)
	if err != nil || role == nil {
		return
	}
	roleName = *role.RoleName
	return
}

func GetIamRoleVersion(roleName string) (version string, err error) {
	svc := connectors.GetAWSSession().IAM
	tagsOutput, err := svc.ListRoleTags(&iam.ListRoleTagsInput{RoleName: &roleName})
	if err != nil {
		return
	}
	for _, tag := range tagsOutput.Tags {
		if *tag.Key == cluster.VersionTagKey {
			version = *tag.Value
			return
		}
	}
	return
}

func GetIamRoleArn(clusterName cluster.ClusterName, roleBaseName string) (arn string, err error) {
	role, err := getIamRole(clusterName, roleBaseName, nil)
	if err != nil || role == nil {
		return
	}
	arn = *role.Arn
	return
}

func UpdateRolePolicy(clusterName cluster.ClusterName, roleBaseName, policyName string, policy PolicyDocument, versionTag []*iam.Tag) error {
	role, err := getIamRole(clusterName, roleBaseName, nil)
	if err != nil {
		return err
	}
	err = removeRolePolicy(role)
	if err != nil {
		return err
	}
	if policy.Version != "" {
		err = attachIamPolicy(*role.RoleName, policyName, policy)
		if err != nil {
			return err
		}
	}
	svc := connectors.GetAWSSession().IAM
	_, err = svc.TagRole(&iam.TagRoleInput{
		RoleName: role.RoleName,
		Tags:     versionTag,
	})
	return err
}

func getRoles(clusterName cluster.ClusterName) (roles []*iam.Role, err error) {
	var marker *string
	isTruncated := true
	var rolesOutput *iam.ListRolesOutput

	log.Debug().Msg("fetching all iam roles ...")

	svc := connectors.GetAWSSession().IAM
	for isTruncated {
		rolesOutput, err = svc.ListRoles(&iam.ListRolesInput{
			Marker:     marker,
			PathPrefix: aws.String(fmt.Sprintf("/wekactl/%s/", clusterName)),
		})
		if err != nil {
			return
		}
		roles = append(roles, rolesOutput.Roles...)
		isTruncated = *rolesOutput.IsTruncated
		marker = rolesOutput.Marker
	}

	return

}

func isClusterRole(role *iam.Role, clusterName cluster.ClusterName) (result bool, err error) {
	svc := connectors.GetAWSSession().IAM
	tagsOutput, err := svc.ListRoleTags(&iam.ListRoleTagsInput{RoleName: role.RoleName})
	if err != nil {
		return
	}
	for _, tag := range tagsOutput.Tags {
		if *tag.Key == cluster.ClusterNameTagKey && *tag.Value == string(clusterName) {
			return true, nil
		}
	}
	return false, nil
}

var tagsSemaphore *semaphore.Weighted

func init() {
	tagsSemaphore = semaphore.NewWeighted(20)
}

func GetClusterRoles(clusterName cluster.ClusterName) (clusterRoles []*iam.Role, err error) {
	var wg sync.WaitGroup
	var responseLock sync.Mutex

	roles, err := getRoles(clusterName)
	if err != nil {
		return
	}

	log.Debug().Msgf("searching for cluster %s roles ...", clusterName)

	wg.Add(len(roles))
	for _, role := range roles {
		go func(role *iam.Role) {
			_ = tagsSemaphore.Acquire(context.Background(), 1)
			defer tagsSemaphore.Release(1)
			defer wg.Done()

			responseLock.Lock()
			defer responseLock.Unlock()
			result, tagsErr := isClusterRole(role, clusterName)
			if tagsErr != nil {
				log.Error().Err(tagsErr)
				log.Error().Msgf("failed to get role %s tags", *role.RoleName)
			}

			if result {
				clusterRoles = append(clusterRoles, role)
			}

		}(role)
	}
	wg.Wait()

	return
}

func DeleteRoles(clusterName cluster.ClusterName, roles []*iam.Role) error {
	for _, role := range roles {
		err := DeleteIamRole(clusterName, *role.RoleName)
		if err != nil {
			return err
		}
	}

	return nil
}
